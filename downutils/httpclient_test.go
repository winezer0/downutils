package downutils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCreateHTTPClient 测试创建HTTP客户端
func TestCreateHTTPClient(t *testing.T) {
	// 测试使用默认配置创建客户端
	client, err := CreateHTTPClient(nil)
	if err != nil {
		t.Fatalf("CreateHTTPClient with nil config failed: %v", err)
	}

	if client == nil {
		t.Fatal("CreateHTTPClient should return non-nil client")
	}

	if client.Transport == nil {
		t.Error("HTTP client should have transport configured")
	}
}

// TestCreateHTTPClientWithConfig 测试使用自定义配置创建HTTP客户端
func TestCreateHTTPClientWithConfig(t *testing.T) {
	config := &HttpClientConfig{
		ConnectTimeout: 15,
		IdleTimeout:    45,
		ProxyURL:       "",
	}

	client, err := CreateHTTPClient(config)
	if err != nil {
		t.Fatalf("CreateHTTPClient with config failed: %v", err)
	}

	if client == nil {
		t.Fatal("CreateHTTPClient should return non-nil client")
	}
}

// TestCreateHTTPClientWithInvalidProxy 测试无效代理URL
func TestCreateHTTPClientWithInvalidProxy(t *testing.T) {
	config := &HttpClientConfig{
		ConnectTimeout: 10,
		IdleTimeout:    30,
		ProxyURL:       "://invalid-proxy-url",
	}

	_, err := CreateHTTPClient(config)
	if err == nil {
		t.Error("CreateHTTPClient should fail with invalid proxy URL")
	}
}

// TestCreateHTTPClientWithHTTPProxy 测试HTTP代理
func TestCreateHTTPClientWithHTTPProxy(t *testing.T) {
	config := &HttpClientConfig{
		ConnectTimeout: 10,
		IdleTimeout:    30,
		ProxyURL:       "http://127.0.0.1:8080",
	}

	client, err := CreateHTTPClient(config)
	if err != nil {
		t.Fatalf("CreateHTTPClient with HTTP proxy failed: %v", err)
	}

	if client == nil {
		t.Fatal("CreateHTTPClient should return non-nil client")
	}
}

// TestCreateHTTPClientWithSOCKS5Proxy 测试SOCKS5代理
func TestCreateHTTPClientWithSOCKS5Proxy(t *testing.T) {
	config := &HttpClientConfig{
		ConnectTimeout: 10,
		IdleTimeout:    30,
		ProxyURL:       "socks5://127.0.0.1:1080",
	}

	client, err := CreateHTTPClient(config)
	if err != nil {
		t.Fatalf("CreateHTTPClient with SOCKS5 proxy failed: %v", err)
	}

	if client == nil {
		t.Fatal("CreateHTTPClient should return non-nil client")
	}
}

// TestDefaultClientConfig 测试默认客户端配置
func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	if config == nil {
		t.Fatal("DefaultClientConfig should return non-nil config")
	}

	if config.ConnectTimeout != 30 {
		t.Errorf("Default ConnectTimeout should be 30, got %d", config.ConnectTimeout)
	}

	if config.IdleTimeout != 60 {
		t.Errorf("Default IdleTimeout should be 60, got %d", config.IdleTimeout)
	}

	if config.ProxyURL != "" {
		t.Errorf("Default ProxyURL should be empty, got %s", config.ProxyURL)
	}
}

// TestHttpGet 测试HTTP GET请求
func TestHttpGet(t *testing.T) {
	// 创建测试服务器
	testContent := "test response"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证User-Agent
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			t.Error("Request should have User-Agent header")
		}

		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 创建context
	ctx := context.Background()

	// 测试GET请求
	resp, err := HttpGet(ctx, client, server.URL)
	if err != nil {
		t.Fatalf("HttpGet failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestHttpGet404 测试HTTP GET请求404错误
func TestHttpGet404(t *testing.T) {
	// 创建返回404的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 创建context
	ctx := context.Background()

	// 测试GET请求
	_, err = HttpGet(ctx, client, server.URL)
	if err == nil {
		t.Error("HttpGet should fail with 404")
	}

	// 验证错误类型
	if !isDownloadError(err, ErrResourceNotFound) {
		t.Errorf("expected RESOURCE_NOT_FOUND error, got: %v", err)
	}
}

// TestHttpGet500 测试HTTP GET请求500错误
func TestHttpGet500(t *testing.T) {
	// 创建返回500的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试GET请求
	_, err = HttpGet(nil, client, server.URL)
	if err == nil {
		t.Error("HttpGet should fail with 500")
	}
}

// TestHttpGetInvalidURL 测试无效URL
func TestHttpGetInvalidURL(t *testing.T) {
	// 创建HTTP客户端
	client, err := CreateHTTPClient(DefaultClientConfig())
	if err != nil {
		t.Fatalf("failed to create HTTP client: %v", err)
	}

	// 测试无效URL
	_, err = HttpGet(nil, client, "://invalid-url")
	if err == nil {
		t.Error("HttpGet should fail with invalid URL")
	}
}
