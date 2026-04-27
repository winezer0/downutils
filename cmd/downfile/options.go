package main

import (
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/winezer0/xutils/logging"
	"os"
)

// 版本信息常量(根据实际情况修改)
const (
	AppName      = "downfile"
	AppShortDesc = "ISEC 依赖文件这下载工具"
	AppLongDesc  = "ISEC 依赖文件这下载工具 根据配置文件进行文件下载"
	AppVersion   = "0.0.3"
	BuildDate    = "2026-04-27"
)

// Options 应用配置结构体
type Options struct {
	ConfigFile string `short:"c" long:"config" description:"配置文件路径" default:"config.yaml"`

	OutputForce string `short:"o" long:"output" description:"强制保存目录 忽略配置文件中的 storage-dir 配置" default:"downloads"`
	UpdateForce bool   `short:"u" long:"update" description:"强制启用更新 忽略配置文件中的 keep-updated 策略"`
	EnableForce bool   `short:"e" long:"enable" description:"强制启用所有下载项 忽略配置文件中 enable 策略"`
	NoProgress  bool   `short:"q" long:"no-bar" description:"不显示下载进度条"`

	ProxyURL string `short:"p" long:"proxy" description:"代理URL 支持http://和socks5://格式" default:""`

	MaxConcurrent  int   `short:"C" long:"concurrent" description:"最大并发下载数" default:"1"`
	MaxSpeed       int64 `short:"S" long:"max-speed" description:"最大下载速度(字节/秒，0表示不限速)" default:"0"`
	ConnectTimeout int   `short:"t" long:"connect-timeout" description:"连接超时时间(秒)" default:"10"`
	IdleTimeout    int   `short:"T" long:"idle-timeout" description:"空闲超时时间(秒)" default:"60"`
	Retries        int   `short:"r" long:"retries" description:"下载失败重试次数" default:"3"`

	Version       bool   `short:"v" long:"version" description:"显示版本信息"`
	LogFile       string `long:"lf" description:"日志文件路径(默认：空)" default:""`
	LogLevel      string `long:"ll" description:"日志级别(debug/info/warn/error)" default:"info"`
	ConsoleFormat string `long:"cf" description:"控制台日志格式(T L C M F组合或off|null禁用)" default:"M"`
}

// InitOptionsArgs 常用的工具函数，解析parser和logging配置
func InitOptionsArgs(minimumParams int) (*Options, *flags.Parser) {
	opts := &Options{}
	parser := flags.NewParser(opts, flags.Default)
	parser.Name = AppName
	parser.Usage = "[OPTIONS]"
	parser.ShortDescription = AppShortDesc
	parser.LongDescription = AppLongDesc

	// 命令行参数数量检查 指不包含程序名本身的参数数量
	if minimumParams > 0 && len(os.Args)-1 < minimumParams {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	// 命令行参数解析检查
	if _, err := parser.Parse(); err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && errors.Is(flagsErr.Type, flags.ErrHelp) {
			os.Exit(0)
		}
		fmt.Printf("Error:%v\n", err)
		os.Exit(1)
	}

	// 版本号输出
	if opts.Version {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	// 初始化日志器
	logCfg := logging.NewLogConfig(opts.LogLevel, opts.LogFile, opts.ConsoleFormat)
	if err := logging.InitDefaultLogger(logCfg); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	if opts.ConfigFile == "" {
		logging.Errorf("config file path is empty")
		os.Exit(1)
	}

	if !fileExists(opts.ConfigFile) {
		logging.Errorf("config file not found: %s", opts.ConfigFile)
	}

	return opts, parser
}

// fileExists 检查文件是否存在
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// displayOptions 显示配置信息(用于日志记录)
func displayOptions(options *Options) {
	logging.Infof("Auto Download Tool %s", AppVersion)
	if options.OutputForce != "" {
		logging.Infof("Force output directory: %s", options.OutputForce)
	}
	logging.Infof("Download Proxy: %s", options.ProxyURL)
	logging.Infof("Connect timeout: %ds", options.ConnectTimeout)
	logging.Infof("Idle timeout: %ds", options.IdleTimeout)
	logging.Infof("Retries: %d", options.Retries)
	logging.Infof("Max concurrent: %d", options.MaxConcurrent)
	if options.MaxSpeed > 0 {
		logging.Infof("Max speed: %d bytes/s", options.MaxSpeed)
	} else {
		logging.Infof("Max speed: unlimited")
	}
	logging.Infof("Force update: %v", options.UpdateForce)
	logging.Infof("Enable force: %v", options.EnableForce)
}
