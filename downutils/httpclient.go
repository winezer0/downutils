package downutils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// CreateHTTPClient 创建HTTP客户端
func CreateHTTPClient(config *HttpClientConfig) (*http.Client, error) {
	// 使用默认配置(如果传入nil)
	if config == nil {
		config = DefaultClientConfig()
	}

	// 设置代理
	var proxyFunc func(*http.Request) (*url.URL, error)
	if config.ProxyURL != "" {
		proxyURL, err := url.Parse(config.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
		}
		proxyFunc = http.ProxyURL(proxyURL)
	} else {
		proxyFunc = http.ProxyFromEnvironment
	}

	// 配置传输设置
	transport := &http.Transport{
		Proxy: proxyFunc,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(config.ConnectTimeout) * time.Second, // 连接超时
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       time.Duration(config.IdleTimeout) * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: time.Duration(config.ConnectTimeout) * time.Second,
	}

	// 创建HTTP客户端
	httpClient := &http.Client{
		Transport: transport,
		// 不设置整体超时，使用context控制
	}

	return httpClient, nil
}

// HttpGet 发送HTTP GET请求
func HttpGet(ctx context.Context, client *http.Client, downloadUrl string) (*http.Response, error) {
	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", downloadUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 设置User-Agent以避免某些服务器的限制
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, classifyError(err, 0)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, classifyError(fmt.Errorf("HTTP request failed"), resp.StatusCode)
	}
	return resp, nil
}

// HttpGetWithRange 发送HTTP GET请求，支持Range头(断点续传)
// rangeStart: 已下载的字节数，为0时不使用Range头
func HttpGetWithRange(ctx context.Context, client *http.Client, downloadUrl string, rangeStart int64) (*http.Response, error) {
	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", downloadUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 设置User-Agent以避免某些服务器的限制
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// 如果已下载部分数据，添加Range头实现断点续传
	if rangeStart > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", rangeStart))
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, classifyError(err, 0)
	}

	// 检查响应状态
	// 200 OK: 完整下载(服务器不支持Range或从头开始)
	// 206 Partial Content: 断点续传成功
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, classifyError(fmt.Errorf("HTTP request failed"), resp.StatusCode)
	}

	return resp, nil
}

// HttpHeadRequest 发送HTTP HEAD请求，用于检查服务器是否支持Range请求
func HttpHeadRequest(ctx context.Context, client *http.Client, downloadUrl string) (*http.Response, error) {
	// 创建HTTP HEAD请求
	req, err := http.NewRequestWithContext(ctx, "HEAD", downloadUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	// 设置User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, classifyError(err, 0)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, classifyError(fmt.Errorf("HEAD request failed"), resp.StatusCode)
	}

	return resp, nil
}
