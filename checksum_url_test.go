package downutils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestResolveChecksumURL(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		body      string
		wantType  ChecksumType
		wantValue string
	}{
		{name: "bare MD5 digest", body: strings.Repeat("a", 32), wantType: ChecksumMD5, wantValue: strings.Repeat("a", 32)},
		{name: "bare SHA1 digest", body: strings.Repeat("b", 40), wantType: ChecksumSHA1, wantValue: strings.Repeat("b", 40)},
		{name: "bare SHA256 digest", body: strings.Repeat("c", 64), wantType: ChecksumSHA256, wantValue: strings.Repeat("c", 64)},
		{
			name:      "manifest matches URL basename and infers algorithm",
			body:      strings.Repeat("d", 64) + "  other.zip\n" + strings.Repeat("e", 40) + " *package.tar.gz\n",
			wantType:  ChecksumSHA1,
			wantValue: strings.Repeat("e", 40),
		},
		{
			name:      "prefixed manifest selects algorithm",
			prefix:    "sha256",
			body:      strings.Repeat("f", 40) + " package.tar.gz\n" + strings.Repeat("a", 64) + " *package.tar.gz\n",
			wantType:  ChecksumSHA256,
			wantValue: strings.Repeat("a", 64),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodGet {
					t.Errorf("checksum request method = %s, want GET", request.Method)
				}
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()

			checksumURL := server.URL + "/checksums"
			if test.prefix != "" {
				checksumURL = test.prefix + ":" + checksumURL
			}
			got, err := resolveChecksum(context.Background(), server.Client(), checksumURL,
				"https://downloads.example/releases/package.tar.gz?download=1")
			if err != nil {
				t.Fatalf("resolveChecksum() error = %v", err)
			}
			if !got.Enable || got.Type != test.wantType || got.Value != test.wantValue {
				t.Fatalf("resolveChecksum() = %#v, want type %s and value %s", got, test.wantType, test.wantValue)
			}
		})
	}
}

func TestResolveChecksumURLFailures(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		statusCode int
		body       string
		wantStatus int
	}{
		{name: "algorithm mismatch", prefix: "sha1", body: strings.Repeat("a", 64)},
		{name: "manifest entry missing", body: strings.Repeat("a", 64) + " other.zip\n"},
		{name: "malformed response", body: "not-a-checksum"},
		{name: "HTTP failure", statusCode: http.StatusBadGateway, body: "unavailable", wantStatus: http.StatusBadGateway},
		{name: "response too large", body: strings.Repeat("x", maxChecksumResponseBytes+1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				statusCode := test.statusCode
				if statusCode == 0 {
					statusCode = http.StatusOK
				}
				writer.WriteHeader(statusCode)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()

			checksumURL := server.URL + "/checksums"
			if test.prefix != "" {
				checksumURL = test.prefix + ":" + checksumURL
			}
			_, err := resolveChecksum(context.Background(), server.Client(), checksumURL,
				"https://downloads.example/releases/package.zip")
			var downloadErr DownloadError
			if !errors.As(err, &downloadErr) || downloadErr.Type != ErrChecksumSourceFailed {
				t.Fatalf("resolveChecksum() error = %v, want %s", err, ErrChecksumSourceFailed)
			}
			if downloadErr.StatusCode != test.wantStatus {
				t.Errorf("error status code = %d, want %d", downloadErr.StatusCode, test.wantStatus)
			}
		})
	}
}

func TestResolveChecksumEmptySkipsRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()

	got, err := resolveChecksum(context.Background(), server.Client(), " ", server.URL+"/package.zip")
	if err != nil {
		t.Fatalf("resolveChecksum() error = %v", err)
	}
	if got.Enable {
		t.Fatalf("resolveChecksum() = %#v, want checksum disabled", got)
	}
	if requests.Load() != 0 {
		t.Fatalf("empty checksum made %d requests, want none", requests.Load())
	}
}

func TestDownloadFileChecksumMismatchDoesNotReplaceTarget(t *testing.T) {
	asset := []byte("new asset contents")
	wrongDigest := sha256.Sum256([]byte("different contents"))
	var assetRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/SHA256SUMS":
			_, _ = fmt.Fprintf(writer, "%s  artifact.zip\n", hex.EncodeToString(wrongDigest[:]))
		case "/artifact.zip":
			assetRequests.Add(1)
			_, _ = writer.Write(asset)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "artifact.zip")
	if err := os.WriteFile(target, []byte("old target contents"), 0600); err != nil {
		t.Fatal(err)
	}
	err := downloadFile(context.Background(), server.Client(), server.URL+"/artifact.zip", target,
		&DownloadTaskConfig{Checksum: "sha256:" + server.URL + "/SHA256SUMS"})
	assertDownloadErrorType(t, err, ErrChecksumMismatch)
	assertFileContents(t, target, "old target contents")
	if assetRequests.Load() != 1 {
		t.Errorf("artifact requests = %d, want 1", assetRequests.Load())
	}
}

func TestDownloadFileChecksumSourceFailureDoesNotReplaceTarget(t *testing.T) {
	var artifactRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/artifact.zip" {
			artifactRequests.Add(1)
		}
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "artifact.zip")
	if err := os.WriteFile(target, []byte("old target contents"), 0600); err != nil {
		t.Fatal(err)
	}
	err := downloadFile(context.Background(), server.Client(), server.URL+"/artifact.zip", target,
		&DownloadTaskConfig{Checksum: "sha256:" + server.URL + "/SHA256SUMS"})
	assertDownloadErrorType(t, err, ErrChecksumSourceFailed)
	assertFileContents(t, target, "old target contents")
	if artifactRequests.Load() != 0 {
		t.Errorf("artifact requests = %d, want 0 after checksum source failure", artifactRequests.Load())
	}
}

func TestDownloadFileWithoutChecksumMakesOnlyArtifactRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(writer, "artifact")
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "artifact.zip")
	if err := downloadFile(context.Background(), server.Client(), server.URL+"/artifact.zip", target, &DownloadTaskConfig{}); err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}
	if requests.Load() != 1 {
		t.Errorf("requests = %d, want only the artifact request", requests.Load())
	}
}

func TestDownloadFileChecksumUsesConfiguredProxy(t *testing.T) {
	asset := []byte("proxied asset")
	digest := sha256.Sum256(asset)
	var mu sync.Mutex
	requests := make(map[string]int)
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Host != "release.example" || request.URL.Scheme != "http" {
			t.Errorf("proxy received URL %q, want http://release.example/...", request.URL.String())
		}
		mu.Lock()
		requests[request.URL.Path]++
		mu.Unlock()
		switch request.URL.Path {
		case "/SHA256SUMS":
			_, _ = fmt.Fprintf(writer, "%s  artifact.zip\n", hex.EncodeToString(digest[:]))
		case "/artifact.zip":
			_, _ = writer.Write(asset)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer proxy.Close()

	client, err := CreateHTTPClient(&HttpClientConfig{ConnectTimeout: 5, IdleTimeout: 30, ProxyURL: proxy.URL})
	if err != nil {
		t.Fatalf("CreateHTTPClient() error = %v", err)
	}
	target := filepath.Join(t.TempDir(), "artifact.zip")
	err = downloadFile(context.Background(), client, "http://release.example/artifact.zip", target,
		&DownloadTaskConfig{Checksum: "sha256:http://release.example/SHA256SUMS"})
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}
	assertFileContents(t, target, string(asset))

	mu.Lock()
	defer mu.Unlock()
	if requests["/SHA256SUMS"] != 1 || requests["/artifact.zip"] != 1 {
		t.Errorf("proxy request counts = %#v, want checksum and artifact once each", requests)
	}
}

func TestExecuteDownloadsWithClientUsesClientForChecksum(t *testing.T) {
	asset := "custom client artifact"
	digest := sha256.Sum256([]byte(asset))
	var requests []string
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request.URL.Path)
		var body string
		switch request.URL.Path {
		case "/SHA256SUMS":
			body = fmt.Sprintf("%s  artifact.zip\n", hex.EncodeToString(digest[:]))
		case "/artifact.zip":
			body = asset
		default:
			return nil, fmt.Errorf("unexpected request URL: %s", request.URL)
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader(body)),
			ContentLength: int64(len(body)),
			Request:       request,
		}, nil
	})}

	storageDir := t.TempDir()
	config := DownConfig{"release": {{
		Module:       "artifact",
		FileName:     "artifact.zip",
		DownloadURLs: []string{"https://release.example/artifact.zip"},
		KeepUpdated:  "enable",
		StorageDir:   storageDir,
		Enable:       true,
		Checksum:     "sha256:https://release.example/SHA256SUMS",
	}}}
	options := NewDefaultDownOptions()
	options.MaxRetries = 1
	options.ShowProgress = false
	result, err := ExecuteDownloadsWithClient(options, config, client)
	if err != nil {
		t.Fatalf("ExecuteDownloadsWithClient() error = %v", err)
	}
	if result.SuccessItems != 1 {
		t.Fatalf("successful items = %d, want 1 (result: %#v)", result.SuccessItems, result)
	}
	assertFileContents(t, filepath.Join(storageDir, "artifact.zip"), asset)
	if len(requests) != 2 || requests[0] != "/SHA256SUMS" || requests[1] != "/artifact.zip" {
		t.Errorf("custom client requests = %#v, want checksum then artifact", requests)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func assertDownloadErrorType(t *testing.T, err error, wantType string) {
	t.Helper()
	var downloadErr DownloadError
	if !errors.As(err, &downloadErr) || downloadErr.Type != wantType {
		t.Fatalf("error = %v, want DownloadError type %s", err, wantType)
	}
}

func assertFileContents(t *testing.T, filePath, want string) {
	t.Helper()
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read %s: %v", filePath, err)
	}
	if string(got) != want {
		t.Fatalf("file %s = %q, want %q", filePath, got, want)
	}
}
