package downutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestInvalidDownloadPreservesExistingTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "invalid") }))
	defer server.Close()
	for _, config := range []DownloadTaskConfig{{CheckSize: "100B"}, {Checksum: "sha256:" + strings.Repeat("0", 64)}} {
		target := filepath.Join(t.TempDir(), "target")
		if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := downloadFile(context.Background(), server.Client(), server.URL, target, &config); err == nil {
			t.Fatal("expected verification failure")
		}
		data, err := os.ReadFile(target)
		if err != nil || string(data) != "original" {
			t.Fatalf("old target changed: %q, %v", data, err)
		}
	}
}

func TestReplaceFileFailurePreservesTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(target+".missing", target); err == nil {
		t.Fatal("expected replacement failure")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "original" {
		t.Fatalf("target changed: %q, %v", data, err)
	}
}

func TestRateLimitWaitRespondsToCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	reader := NewLimitReaderContext(ctx, &notifyingReader{Reader: strings.NewReader(strings.Repeat("x", 1024)), started: started}, 1)
	done := make(chan error, 1)
	go func() { _, err := reader.Read(make([]byte, 1024)); done <- err }()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("rate-limit wait did not cancel")
	}
}

type notifyingReader struct {
	io.Reader
	started chan struct{}
}

func (r *notifyingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	close(r.started)
	return n, err
}

func TestReadIdleTimeoutResetsOnIncomingData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 4; i++ {
			select {
			case <-time.After(400 * time.Millisecond):
			case <-r.Context().Done():
				return
			}
			io.WriteString(w, "x")
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := downloadFile(ctx, server.Client(), server.URL, filepath.Join(t.TempDir(), "target"), &DownloadTaskConfig{ReadIdleTimeout: 1})
	if err != nil {
		t.Fatalf("active transfer exceeded idle timeout: %v", err)
	}
}

func TestReadIdleTimeoutStopsStalledResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	start := time.Now()
	err := downloadFile(ctx, server.Client(), server.URL, filepath.Join(t.TempDir(), "target"), &DownloadTaskConfig{ReadIdleTimeout: 1})
	var downloadErr DownloadError
	if !errors.As(err, &downloadErr) || downloadErr.Type != ErrTimeout {
		t.Fatalf("expected timeout, got %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("idle timeout did not stop the stalled read")
	}
}

func TestSuccessfulResumeRecordedBeforeRename(t *testing.T) {
	var gets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Accept-Ranges", "bytes")
			return
		}
		if gets.Add(1) == 1 {
			w.Header().Set("Content-Length", "6")
			io.WriteString(w, "abc")
			return
		}
		if r.Header.Get("Range") != "bytes=3-" {
			t.Errorf("unexpected range: %s", r.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", "bytes 3-5/6")
		w.WriteHeader(http.StatusPartialContent)
		io.WriteString(w, "def")
	}))
	defer server.Close()
	options := NewDefaultDownOptions()
	options.ShowProgress = false
	options.MaxRetries = 2
	item := DownItem{Module: "test", FileName: "target", StorageDir: t.TempDir(), DownloadURLs: []string{server.URL}}
	result := processSingleItem(context.Background(), server.Client(), item, options)
	if result.Status != TaskStatusSuccess || !result.Resumed {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, err := os.ReadFile(result.StoragePath)
	if err != nil || string(data) != "abcdef" {
		t.Fatalf("unexpected data: %q, %v", data, err)
	}
}

func TestBothEntrypointsRejectInvalidConfiguration(t *testing.T) {
	config := DownConfig{"test": {{Module: "test", FileName: "target", Enable: true, MaxSize: -1, DownloadURLs: []string{"http://localhost/test"}}}}
	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprint(custom), func(t *testing.T) {
			var err error
			if custom {
				_, err = ExecuteDownloadsWithClientContext(context.Background(), nil, config, http.DefaultClient)
			} else {
				_, err = ExecuteDownloadsContext(context.Background(), nil, config)
			}
			if err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestCancellationDoesNotTryRemainingSources(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); cancel(); <-r.Context().Done() }))
	defer server.Close()
	options := NewDefaultDownOptions()
	options.ShowProgress = false
	items := []DownItem{{Module: "test", FileName: "target", StorageDir: t.TempDir(), DownloadURLs: []string{server.URL, server.URL + "/mirror"}}}
	result := processDownItems(ctx, server.Client(), items, options)
	if len(result) != 1 || result[0].Status != TaskStatusFailed || requests.Load() != 1 {
		t.Fatalf("unexpected cancellation: %+v, requests=%d", result, requests.Load())
	}
}
