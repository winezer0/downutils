package downutils

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecuteDownloadsContextCancelsInFlightRequest(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	}))
	defer server.Close()

	root := t.TempDir()
	config := DownConfig{"osv": {{
		Module: "npm", FileName: "all.zip", DownloadURLs: []string{server.URL},
		StorageDir: root, Enable: true,
	}}}
	options := NewDefaultDownOptions()
	options.ShowProgress = false
	options.MaxRetries = 3
	options.IdleTimeout = 30
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan *DownloadResult, 1)
	go func() {
		result, err := ExecuteDownloadsContext(ctx, options, config)
		if err != nil {
			t.Errorf("ExecuteDownloadsContext() error = %v", err)
		}
		done <- result
	}()
	<-started
	startedAt := time.Now()
	cancel()
	select {
	case result := <-done:
		if elapsed := time.Since(startedAt); elapsed > time.Second {
			t.Fatalf("cancellation took %v", elapsed)
		}
		if result == nil || result.FailedItems != 1 || result.TaskInfos[0].ErrorMsg == "" {
			t.Fatalf("cancellation result = %#v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("download did not stop after context cancellation")
	}
}

// TestDownloadFileRestartsPartialData checks full-download fallbacks and mutable URLs.
func TestDownloadFileRestartsPartialData(t *testing.T) {
	for _, test := range []struct {
		name          string
		headStatus    int
		acceptRanges  string
		disableResume bool
	}{
		{name: "HEAD unsupported", headStatus: http.StatusMethodNotAllowed},
		{name: "ranges unsupported", headStatus: http.StatusOK},
		{name: "range ignored", headStatus: http.StatusOK, acceptRanges: "bytes"},
		{name: "resume disabled", headStatus: http.StatusOK, disableResume: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method == http.MethodHead {
					writer.Header().Set("Accept-Ranges", test.acceptRanges)
					writer.WriteHeader(test.headStatus)
					return
				}
				_, _ = writer.Write([]byte("new-content"))
			}))
			defer server.Close()
			target := filepath.Join(t.TempDir(), "all.zip")
			if err := os.WriteFile(generateTempFileName(target, server.URL), []byte("old"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := downloadFile(context.Background(), server.Client(), server.URL, target, &DownloadTaskConfig{DisableResume: test.disableResume}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(target)
			if err != nil || string(data) != "new-content" {
				t.Fatalf("downloaded content = %q, %v", data, err)
			}
		})
	}
}

// TestDownloadFileLimitsChunkedBody checks the size limit without a Content-Length header.
func TestDownloadFileLimitsChunkedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.(http.Flusher).Flush()
		_, _ = writer.Write([]byte("123456"))
	}))
	defer server.Close()
	target := filepath.Join(t.TempDir(), "all.zip")
	err := downloadFile(context.Background(), server.Client(), server.URL, target, &DownloadTaskConfig{MaxSize: 3})
	var downloadErr DownloadError
	if !errors.As(err, &downloadErr) || downloadErr.Type != ErrMaximumSizeExceeded {
		t.Fatalf("oversized chunked response error = %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("oversized chunked response created final file: %v", err)
	}
}

// TestClassifyContextErrors distinguishes retryable deadlines from explicit cancellation.
func TestClassifyContextErrors(t *testing.T) {
	for _, test := range []struct {
		err  error
		kind string
	}{{context.Canceled, ErrDownloadCancelled}, {context.DeadlineExceeded, ErrTimeout}} {
		var got DownloadError
		if err := classifyError(test.err, 0); !errors.As(err, &got) || got.Type != test.kind {
			t.Fatalf("context error %v classified as %v, want %s", test.err, err, test.kind)
		}
	}
}

func TestExecuteDownloadsContextEnforcesMaximumSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", "6")
		_, _ = writer.Write([]byte("123456"))
	}))
	defer server.Close()

	root := t.TempDir()
	config := DownConfig{"osv": {{
		Module: "npm", FileName: "all.zip", DownloadURLs: []string{server.URL},
		StorageDir: root, Enable: true, MaxSize: 5,
	}}}
	options := NewDefaultDownOptions()
	options.ShowProgress = false
	options.MaxRetries = 1
	result, err := ExecuteDownloadsContext(context.Background(), options, config)
	if err != nil {
		t.Fatal(err)
	}
	if result.FailedItems != 1 || result.TaskInfos[0].ErrorMsg == "" {
		t.Fatalf("oversized download result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "all.zip")); !os.IsNotExist(err) {
		t.Fatalf("oversized download created final file: %v", err)
	}
}
