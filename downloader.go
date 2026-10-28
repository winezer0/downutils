package downutils

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/winezer0/slogs"
)

// TaskStatus 任务状态枚举
type TaskStatus string

const (
	TaskStatusSuccess   TaskStatus = "success"    // 下载成功
	TaskStatusSkipped   TaskStatus = "skipped"    // 跳过(不需要更新)
	TaskStatusFailed    TaskStatus = "failed"     // 下载失败
	TaskStatusConfigErr TaskStatus = "config_err" // 配置错误
)

// TaskInfo 任务详细信息
type TaskInfo struct {
	Module      string     `json:"module"`      // 任务模块名称
	FileName    string     `json:"fileName"`    // 文件名
	StoragePath string     `json:"storagePath"` // 完整存储路径
	DownloadURL string     `json:"downloadUrl"` // 实际使用的下载URL
	Status      TaskStatus `json:"status"`      // 任务状态
	Resumed     bool       `json:"resumed"`     // 是否断点续传
	ErrorMsg    string     `json:"errorMsg"`    // 错误原因(成功时为空)
}

// DownloadResult 下载结果统计
type DownloadResult struct {
	TotalItems   int        `json:"totalItems"`   // 总下载项数
	SuccessItems int        `json:"successItems"` // 成功下载项数
	FailedItems  int        `json:"failedItems"`  // 失败下载项数
	SkippedItems int        `json:"skippedItems"` // 跳过下载项数
	TaskInfos    []TaskInfo `json:"taskInfos"`    // 每个任务的详细信息
}

// ExecuteDownloads 执行完整的下载流程
// 参数说明：
//   - config: 下载配置选项
//   - downConfig: 已加载的下载配置(由调用方负责加载)
func ExecuteDownloads(options *DownOptions, downConfig DownConfig) (*DownloadResult, error) {
	return ExecuteDownloadsContext(context.Background(), options, downConfig)
}

// ExecuteDownloadsContext executes the configured downloads until completion or cancellation.
func ExecuteDownloadsContext(ctx context.Context, options *DownOptions, downConfig DownConfig) (*DownloadResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if options == nil {
		options = NewDefaultDownOptions()
	}

	// 创建HTTP客户端配置
	clientConfig := &HttpClientConfig{
		ConnectTimeout: options.ConnectTimeout,
		IdleTimeout:    options.IdleTimeout,
		ProxyURL:       options.ProxyURL,
	}

	// 创建HTTP客户端
	httpClient, err := CreateHTTPClient(clientConfig)
	if err != nil {
		return nil, err
	}
	defer httpClient.CloseIdleConnections()

	return ExecuteDownloadsWithClientContext(ctx, options, downConfig, httpClient)
}

// ExecuteDownloadsWithClient 使用已有的HTTP客户端执行下载流程
// 适用于需要复用HTTP客户端的场景
func ExecuteDownloadsWithClient(options *DownOptions, downConfig DownConfig, httpClient *http.Client) (*DownloadResult, error) {
	return ExecuteDownloadsWithClientContext(context.Background(), options, downConfig, httpClient)
}

// ExecuteDownloadsWithClientContext executes downloads with a caller-provided HTTP client and context.
func ExecuteDownloadsWithClientContext(ctx context.Context, options *DownOptions, downConfig DownConfig, httpClient *http.Client) (*DownloadResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if options == nil {
		options = NewDefaultDownOptions()
	}
	if httpClient == nil {
		return nil, fmt.Errorf("HTTP client must not be nil")
	}
	if err := validateDownloadOptions(options); err != nil {
		return nil, err
	}
	if errs := ValidateDownConfig(downConfig, options.EnableForce); len(errs) > 0 {
		return nil, fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}
	CheckDuplicatePaths(downConfig, options.OutputForce)

	// 下载前清理所有配置目录中的临时文件
	cleanupAllIncompleteDownloads(downConfig, options.OutputForce)

	// 处理所有配置组
	result := processAllConfigGroups(ctx, httpClient, downConfig, options)

	// 下载后再次清理未完成的下载文件
	cleanupAllIncompleteDownloads(downConfig, options.OutputForce)

	return result, nil
}

func validateDownloadOptions(options *DownOptions) error {
	if options.MaxRetries < 1 {
		return fmt.Errorf("maximum attempts must be at least 1")
	}
	if options.MaxConcurrent < 0 {
		return fmt.Errorf("maximum concurrency must not be negative (0 means serial)")
	}
	if options.MaxSpeed < 0 {
		return fmt.Errorf("maximum speed must not be negative")
	}
	if options.ConnectTimeout < 0 || options.IdleTimeout < 0 || options.ReadIdleTimeout < 0 {
		return fmt.Errorf("timeouts must not be negative")
	}
	return nil
}

// processAllConfigGroups 处理所有配置组的下载任务
func processAllConfigGroups(ctx context.Context, httpClient *http.Client, config DownConfig, options *DownOptions) *DownloadResult {
	result := &DownloadResult{}

	for groupName, downItems := range config {
		// 如果未启用enable过滤，则只处理enable=true的项
		if !options.EnableForce {
			downItems = filterEnableItems(downItems)
		}
		if len(downItems) > 0 {
			slogs.Infof("Processing config group: %s", groupName)
			taskInfos := processDownItems(ctx, httpClient, downItems, options)
			result.TotalItems += len(downItems)
			result.TaskInfos = append(result.TaskInfos, taskInfos...)

			// 统计各状态数量
			for _, info := range taskInfos {
				switch info.Status {
				case TaskStatusSuccess:
					result.SuccessItems++
				case TaskStatusSkipped:
					result.SkippedItems++
				case TaskStatusFailed, TaskStatusConfigErr:
					result.FailedItems++
				}
			}
		}
	}

	return result
}

// DisplayDownloadResult 显示下载结果统计信息
func DisplayDownloadResult(result *DownloadResult) {
	if result == nil {
		slogs.Warnf("download result is nil")
		return
	}

	slogs.Infof("Download completed: %d/%d items successful, %d skipped, %d failed",
		result.SuccessItems, result.TotalItems, result.SkippedItems, result.FailedItems)

	// 显示每个任务的详细信息
	for _, info := range result.TaskInfos {
		switch info.Status {
		case TaskStatusSuccess:
			if info.Resumed {
				slogs.Infof("[SUCCESS] %s (%s) - resumed download", info.Module, info.FileName)
			} else {
				slogs.Infof("[SUCCESS] %s (%s)", info.Module, info.FileName)
			}
		case TaskStatusSkipped:
			slogs.Infof("[SKIPPED] %s (%s) - file exists and no update needed", info.Module, info.FileName)
		case TaskStatusFailed:
			slogs.Errorf("[FAILED] %s (%s) - %s", info.Module, info.FileName, info.ErrorMsg)
		case TaskStatusConfigErr:
			slogs.Errorf("[CONFIG_ERR] %s (%s) - %s", info.Module, info.FileName, info.ErrorMsg)
		}
	}
}
