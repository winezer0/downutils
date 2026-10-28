package downutils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestConcurrentDownloads 测试并发下载
func TestConcurrentDownloads(t *testing.T) {
	// 创建测试服务器
	testContent := "concurrent download test content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_concurrent_downloads")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 并发下载多个文件
	numDownloads := 5
	var wg sync.WaitGroup
	errors := make(chan error, numDownloads)

	for i := 0; i < numDownloads; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			storePath := filepath.Join(tmpDir, fmt.Sprintf("file_%d.txt", index))
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := downloadFile(ctx, client, server.URL, storePath, &DownloadTaskConfig{ShowProgress: false})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errors)

	// 检查错误
	for err := range errors {
		t.Errorf("concurrent download failed: %v", err)
	}

	// 验证所有文件都已下载
	for i := 0; i < numDownloads; i++ {
		storePath := filepath.Join(tmpDir, fmt.Sprintf("file_%d.txt", i))
		if !fileExists(storePath) {
			t.Errorf("downloaded file should exist: %s", storePath)
		}
	}
}

// TestConcurrentProcessDownItems 测试并发处理配置组
func TestConcurrentProcessDownItems(t *testing.T) {
	// 创建测试服务器
	testContent := "concurrent process test"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "test_concurrent_process")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建多组配置
	configs := [][]DownItem{
		{
			{
				Module:       "group1-module1",
				FileName:     "g1_m1.txt",
				DownloadURLs: []string{server.URL},
				Enable:       true,
				StorageDir:   tmpDir,
			},
		},
		{
			{
				Module:       "group2-module1",
				FileName:     "g2_m1.txt",
				DownloadURLs: []string{server.URL},
				Enable:       true,
				StorageDir:   tmpDir,
			},
		},
		{
			{
				Module:       "group3-module1",
				FileName:     "g3_m1.txt",
				DownloadURLs: []string{server.URL},
				Enable:       true,
				StorageDir:   tmpDir,
			},
		},
	}

	// 创建下载选项
	options := &DownOptions{
		OutputForce:    "",
		UpdateForce:    false,
		EnableForce:    false,
		MaxRetries:     1,
		ConnectTimeout: 10,
		IdleTimeout:    30,
	}

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 并发处理多组配置
	var wg sync.WaitGroup
	totalSuccess := 0
	var mu sync.Mutex

	for _, items := range configs {
		wg.Add(1)
		go func(downItems []DownItem) {
			defer wg.Done()

			taskInfos := ProcessDownItems(client, downItems, options)

			// 统计成功任务数
			successCount := 0
			for _, info := range taskInfos {
				if info.Status == TaskStatusSuccess {
					successCount++
				}
			}

			mu.Lock()
			totalSuccess += successCount
			mu.Unlock()
		}(items)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 验证成功数量
	if totalSuccess != len(configs) {
		t.Errorf("expected %d successful downloads, got %d", len(configs), totalSuccess)
	}

	// 验证所有文件都已下载
	for _, items := range configs {
		for _, item := range items {
			storePath := filepath.Join(tmpDir, item.FileName)
			if !fileExists(storePath) {
				t.Errorf("downloaded file should exist: %s", storePath)
			}
		}
	}
}
