package main

import (
	"github.com/winezer0/xutils/logging"
	"os"

	"github.com/winezer0/downutils/downutils"
)

func main() {
	// 打印命令行输入配置
	opts, _ := InitOptionsArgs(1)
	defer logging.Sync()

	// 加载并验证配置文件
	downConfig, err := downutils.LoadConfig(opts.ConfigFile)
	if err != nil {
		logging.Errorf("failed to load config: %v", err)
		os.Exit(1)
	}

	// 显示配置信息
	displayOptions(opts)

	// 构建下载选项
	downOptions := &downutils.DownOptions{
		OutputForce:    opts.OutputForce,
		UpdateForce:    opts.UpdateForce,
		EnableForce:    opts.EnableForce,
		ShowProgress:   !opts.NoProgress,
		MaxRetries:     opts.Retries,
		MaxConcurrent:  opts.MaxConcurrent,
		MaxSpeed:       opts.MaxSpeed,
		ConnectTimeout: opts.ConnectTimeout,
		IdleTimeout:    opts.IdleTimeout,
		ProxyURL:       opts.ProxyURL,
	}

	// 执行下载流程
	result, err := downutils.ExecuteDownloads(downOptions, downConfig)
	if err != nil {
		logging.Errorf("download failed: %v", err)
		os.Exit(1)
	}

	// 显示下载结果
	downutils.DisplayDownloadResult(result)
}
