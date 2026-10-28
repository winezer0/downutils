package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/winezer0/downutils"
	"github.com/winezer0/slogs"
)

func main() {
	os.Exit(run())
}

func run() int {
	// 打印命令行输入配置
	opts, _ := InitOptionsArgs(1)
	defer slogs.CloseAll()

	// 加载并验证配置文件
	downConfig, err := downutils.LoadConfig(opts.ConfigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		return 1
	}

	// 显示配置信息
	displayOptions(opts)

	// 构建下载选项
	downOptions := &downutils.DownOptions{
		DisableResume:   opts.DisableResume,
		ReadIdleTimeout: opts.ReadIdleTimeout,
		OutputForce:     opts.OutputForce,
		UpdateForce:     opts.UpdateForce,
		EnableForce:     opts.EnableForce,
		ShowProgress:    !opts.NoProgress,
		MaxRetries:      opts.Retries,
		MaxConcurrent:   opts.MaxConcurrent,
		MaxSpeed:        opts.MaxSpeed,
		ConnectTimeout:  opts.ConnectTimeout,
		IdleTimeout:     opts.IdleTimeout,
		ProxyURL:        opts.ProxyURL,
	}

	// 执行下载流程
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := downutils.ExecuteDownloadsContext(ctx, downOptions, downConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\n", err)
		return 1
	}

	// 显示下载结果
	downutils.DisplayDownloadResult(result)
	if result != nil && result.FailedItems > 0 {
		fmt.Fprintf(os.Stderr, "Download failed: %d of %d items failed\n", result.FailedItems, result.TotalItems)
	}
	return resultExitCode(ctx, result)
}

func resultExitCode(ctx context.Context, result *downutils.DownloadResult) int {
	if ctx.Err() != nil {
		return 130
	}
	if result == nil || result.FailedItems > 0 {
		return 1
	}
	return 0
}
