package downutils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDownloadFile 测试下载文件功能
func TestDownloadFile(t *testing.T) {
	// 创建测试服务器
	testContent := "test file content for download"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "30")
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_download")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试下载文件
	storePath := filepath.Join(tmpDir, "test.txt")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = downloadFile(ctx, client, server.URL, storePath, &DownloadTaskConfig{ShowProgress: true})
	if err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}

	// 验证文件内容
	content, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("downloaded content mismatch: got %s, want %s", string(content), testContent)
	}
}

// TestDownloadFileOverwrite 测试覆盖已存在文件
func TestDownloadFileOverwrite(t *testing.T) {
	// 创建测试服务器
	testContent := "new content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_download_overwrite")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建旧文件
	storePath := filepath.Join(tmpDir, "test.txt")
	oldContent := "old content"
	os.WriteFile(storePath, []byte(oldContent), 0644)

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试下载文件(应该覆盖旧文件)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = downloadFile(ctx, client, server.URL, storePath, &DownloadTaskConfig{ShowProgress: true})
	if err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}

	// 验证旧文件已被删除
	oldFilePath := storePath + ".old"
	if fileExists(oldFilePath) {
		t.Error("old file should be deleted, not backed up")
	}

	// 验证新文件内容
	content, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("downloaded content mismatch: got %s, want %s", string(content), testContent)
	}
}

// TestDownloadFileContextCancel 测试context取消
func TestDownloadFileContextCancel(t *testing.T) {
	// 创建慢速测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000000")
		// 写入少量数据后休眠
		w.Write([]byte("some data"))
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_download_cancel")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 创建短超时context
	storePath := filepath.Join(tmpDir, "test.txt")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 测试下载应该被取消
	err = downloadFile(ctx, client, server.URL, storePath, &DownloadTaskConfig{ShowProgress: true})
	if err == nil {
		t.Error("downloadFile should fail when context is cancelled")
	}

	// 验证错误类型
	if !isDownloadError(err, ErrDownloadCancelled) && ctx.Err() == nil {
		t.Errorf("expected cancellation error, got: %v", err)
	}
}

// TestDownloadFile404 测试404错误
func TestDownloadFile404(t *testing.T) {
	// 创建返回404的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_download_404")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试下载应该返回404错误
	storePath := filepath.Join(tmpDir, "test.txt")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = downloadFile(ctx, client, server.URL, storePath, &DownloadTaskConfig{ShowProgress: true})
	if err == nil {
		t.Error("downloadFile should fail with 404")
	}

	// 验证错误类型
	if !isDownloadError(err, ErrResourceNotFound) {
		t.Errorf("expected RESOURCE_NOT_FOUND error, got: %v", err)
	}
}

// TestDownloadFileUnknownSize 测试未知文件大小
func TestDownloadFileUnknownSize(t *testing.T) {
	// 创建不返回Content-Length的测试服务器
	testContent := "test content without content-length"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 不设置Content-Length
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_download_unknown_size")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试下载文件
	storePath := filepath.Join(tmpDir, "test.txt")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = downloadFile(ctx, client, server.URL, storePath, &DownloadTaskConfig{ShowProgress: true})
	if err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}

	// 验证文件内容
	content, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("downloaded content mismatch: got %s, want %s", string(content), testContent)
	}
}

// isDownloadError 检查错误是否为指定类型的DownloadError
func isDownloadError(err error, errorType string) bool {
	if err != nil {
		if downloadErr, ok := err.(DownloadError); ok {
			return downloadErr.Type == errorType
		}
	}
	return false
}
