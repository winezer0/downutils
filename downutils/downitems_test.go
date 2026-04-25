package downutils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestProcessDownItems 测试处理配置组
func TestProcessDownItems(t *testing.T) {
	// 创建测试服务器
	testContent := "test content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_process_items")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试项
	items := []DownItem{
		{
			Module:       "test-module",
			FileName:     "test.txt",
			DownloadURLs: []string{server.URL},
			KeepUpdated:  "enable",
			Enable:       true,
			StorageDir:   tmpDir,
		},
	}

	// 创建下载选项
	options := &DownOptions{
		OutputForce:    "",
		UpdateForce:    false,
		MaxRetries:     1,
		ConnectTimeout: 10,
		IdleTimeout:    30,
	}

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试处理配置组
	taskInfos := ProcessDownItems(client, items, options)

	if len(taskInfos) != 1 || taskInfos[0].Status != TaskStatusSuccess {
		t.Errorf("ProcessDownItems should succeed 1 item, got %d tasks", len(taskInfos))
	}

	// 验证文件已下载
	storePath := filepath.Join(tmpDir, "test.txt")
	if !fileExists(storePath) {
		t.Error("downloaded file should exist")
	}
}

// TestProcessDownItemsMultipleURLs 测试多URL下载
func TestProcessDownItemsMultipleURLs(t *testing.T) {
	// 创建第一个失败的服务器
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server1.Close()

	// 创建第二个成功的服务器
	testContent := "success content"
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server2.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_process_multi_urls")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试项(第一个URL失败，第二个成功)
	items := []DownItem{
		{
			Module:       "test-multi-url",
			FileName:     "multi.txt",
			DownloadURLs: []string{server1.URL, server2.URL},
			KeepUpdated:  "enable",
			Enable:       true,
			StorageDir:   tmpDir,
		},
	}

	// 创建下载选项
	options := &DownOptions{
		OutputForce:    "",
		UpdateForce:    false,
		MaxRetries:     1,
		ConnectTimeout: 10,
		IdleTimeout:    30,
	}

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试处理配置组
	taskInfos := ProcessDownItems(client, items, options)

	if len(taskInfos) != 1 || taskInfos[0].Status != TaskStatusSuccess {
		t.Errorf("ProcessDownItems should succeed 1 item with fallback URL, got %d tasks", len(taskInfos))
	}
}

// TestProcessDownItemsSkipExisting 测试跳过已存在文件
func TestProcessDownItemsSkipExisting(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_process_skip")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建已存在的文件
	storePath := filepath.Join(tmpDir, "existing.txt")
	os.WriteFile(storePath, []byte("existing content"), 0644)

	// 创建测试项
	items := []DownItem{
		{
			Module:       "existing-module",
			FileName:     "existing.txt",
			DownloadURLs: []string{"http://invalid-url-that-should-not-be-called.com"},
			KeepUpdated:  "disable",
			Enable:       true,
			StorageDir:   tmpDir,
		},
	}

	// 创建下载选项(不更新已存在文件)
	options := &DownOptions{
		OutputForce:    "",
		UpdateForce:    false,
		MaxRetries:     1,
		ConnectTimeout: 10,
		IdleTimeout:    30,
	}

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试处理配置组(应该跳过)
	taskInfos := ProcessDownItems(client, items, options)

	if len(taskInfos) != 1 || taskInfos[0].Status != TaskStatusSkipped {
		t.Errorf("ProcessDownItems should skip existing file and return 1 skipped task, got %d tasks", len(taskInfos))
	}
}

// TestProcessDownItemsForceUpdate 测试强制更新功能
func TestProcessDownItemsForceUpdate(t *testing.T) {
	// 创建测试服务器
	testContent := "force update content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_process_force_update")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建已存在的文件
	storePath := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(storePath, []byte("old content"), 0644)

	// 创建测试项(设置为 disable 更新策略)
	items := []DownItem{
		{
			Module:       "force-update-module",
			FileName:     "test.txt",
			DownloadURLs: []string{server.URL},
			KeepUpdated:  "disable",
			Enable:       true,
			StorageDir:   tmpDir,
		},
	}

	// 创建下载选项(启用强制更新)
	options := &DownOptions{
		OutputForce:    "",
		UpdateForce:    true,
		MaxRetries:     1,
		ConnectTimeout: 10,
		IdleTimeout:    30,
	}

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试处理配置组(应该强制更新)
	taskInfos := ProcessDownItems(client, items, options)

	if len(taskInfos) != 1 || taskInfos[0].Status != TaskStatusSuccess {
		t.Errorf("ProcessDownItems with force update should succeed 1 item, got %d tasks", len(taskInfos))
	}

	// 验证文件内容已更新
	content, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("downloaded content mismatch: got %s, want %s", string(content), testContent)
	}
}

// TestProcessDownItemsWithStorageDir 测试自定义存储目录
func TestProcessDownItemsWithStorageDir(t *testing.T) {
	// 创建测试服务器
	testContent := "custom dir content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_process_custom_dir")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 自定义存储目录
	customDir := filepath.Join(tmpDir, "custom")

	// 创建测试项
	items := []DownItem{
		{
			Module:       "custom-dir-module",
			FileName:     "custom.txt",
			DownloadURLs: []string{server.URL},
			KeepUpdated:  "enable",
			Enable:       true,
			StorageDir:   customDir,
		},
	}

	// 创建下载选项
	options := &DownOptions{
		OutputForce:    "",
		UpdateForce:    false,
		MaxRetries:     1,
		ConnectTimeout: 10,
		IdleTimeout:    30,
	}

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试处理配置组
	taskInfos := ProcessDownItems(client, items, options)

	if len(taskInfos) != 1 || taskInfos[0].Status != TaskStatusSuccess {
		t.Errorf("ProcessDownItems should succeed 1 item, got %d tasks", len(taskInfos))
	}

	// 验证文件在自定义目录
	storePath := filepath.Join(customDir, "custom.txt")
	if !fileExists(storePath) {
		t.Errorf("file should exist in custom directory: %s", storePath)
	}
}

// TestProcessDownItemsNilOptions 测试nil选项
func TestProcessDownItemsNilOptions(t *testing.T) {
	// 创建测试服务器
	testContent := "nil options content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_process_nil_opts")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建测试项
	items := []DownItem{
		{
			Module:       "nil-opts-module",
			FileName:     "nil_opts.txt",
			DownloadURLs: []string{server.URL},
			KeepUpdated:  "",
			Enable:       true,
			StorageDir:   tmpDir,
		},
	}

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试处理配置组(使用nil选项)
	taskInfos := ProcessDownItems(client, items, nil)

	if len(taskInfos) != 1 || taskInfos[0].Status != TaskStatusSuccess {
		t.Errorf("ProcessDownItems with nil options should succeed 1 item, got %d tasks", len(taskInfos))
	}
}

// TestProcessDownItemsWithOutputForce 测试OutputForce强制保存目录
func TestProcessDownItemsWithOutputForce(t *testing.T) {
	// 创建测试服务器
	testContent := "output force content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_process_output_force")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建自定义存储目录和强制保存目录
	customDir := filepath.Join(tmpDir, "custom")
	forceDir := filepath.Join(tmpDir, "force")

	// 创建测试项(配置了StorageDir)
	items := []DownItem{
		{
			Module:       "output-force-module",
			FileName:     "force.txt",
			DownloadURLs: []string{server.URL},
			KeepUpdated:  "enable",
			Enable:       true,
			StorageDir:   customDir,
		},
	}

	// 创建下载选项(启用OutputForce，优先级应高于StorageDir)
	options := &DownOptions{
		OutputForce:    forceDir,
		UpdateForce:    false,
		MaxRetries:     1,
		ConnectTimeout: 10,
		IdleTimeout:    30,
	}

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试处理配置组
	taskInfos := ProcessDownItems(client, items, options)

	if len(taskInfos) != 1 || taskInfos[0].Status != TaskStatusSuccess {
		t.Errorf("ProcessDownItems should succeed 1 item, got %d tasks", len(taskInfos))
	}

	// 验证文件在强制保存目录，而不是StorageDir
	forcePath := filepath.Join(forceDir, "force.txt")
	customPath := filepath.Join(customDir, "force.txt")

	if !fileExists(forcePath) {
		t.Errorf("file should exist in force directory: %s", forcePath)
	}

	if fileExists(customPath) {
		t.Errorf("file should NOT exist in custom directory when OutputForce is set: %s", customPath)
	}
}
