package main

import (
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/winezer0/slogs"
	"os"
)

// 版本信息常量(根据实际情况修改)
const (
	AppName      = "downfile"
	AppShortDesc = "ISEC dependency downloader"
	AppLongDesc  = "Download dependency files from a YAML configuration."
	AppVersion   = "0.0.3"
	BuildDate    = "2026-04-27"
)

// Options 应用配置结构体
type Options struct {
	ConfigFile string `short:"c" long:"config" description:"Configuration file path" default:"config.yaml"`

	OutputForce   string `short:"o" long:"output" description:"Override configured storage directories" default:"downloads"`
	UpdateForce   bool   `short:"u" long:"update" description:"Force updates regardless of keep-updated policy"`
	EnableForce   bool   `short:"e" long:"enable" description:"Enable all configured download items"`
	NoProgress    bool   `short:"q" long:"no-bar" description:"Disable progress display"`
	DisableResume bool   `long:"disable-resume" description:"Restart each attempt without resuming partial data"`

	ProxyURL string `short:"p" long:"proxy" description:"Proxy URL (http:// or socks5://)" default:""`

	MaxConcurrent   int   `short:"C" long:"concurrent" description:"Maximum concurrent downloads (at least 1)" default:"1"`
	MaxSpeed        int64 `short:"S" long:"max-speed" description:"Bytes per second per download (0: unlimited)" default:"0"`
	ConnectTimeout  int   `short:"t" long:"connect-timeout" description:"Connection and response-header timeout in seconds" default:"10"`
	IdleTimeout     int   `short:"T" long:"idle-timeout" description:"Legacy total timeout per attempt and idle connection timeout in seconds" default:"60"`
	ReadIdleTimeout int   `long:"read-idle-timeout" description:"Response-body read idle timeout in seconds (0: disabled)" default:"0"`
	Retries         int   `short:"r" long:"retries" description:"Maximum attempts per URL, including the first attempt" default:"3"`

	Version       bool   `short:"v" long:"version" description:"Show version information"`
	LogFile       string `long:"lf" description:"Optional log file path" default:""`
	LogLevel      string `long:"ll" description:"Log level (debug/info/warn/error)" default:"info"`
	ConsoleFormat string `long:"cf" description:"Console fields (T L C M F; off|null: disabled)" default:"M"`
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 版本号输出
	if opts.Version {
		fmt.Printf("%s version %s\n", AppName, AppVersion)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	// 初始化日志器
	logCfg := slogs.NewConfig(opts.LogLevel, opts.LogFile, opts.ConsoleFormat)
	if err := slogs.Init(logCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	if opts.ConfigFile == "" {
		fmt.Fprintln(os.Stderr, "Config file path is empty")
		os.Exit(1)
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
	slogs.Infof("Auto Download Tool %s", AppVersion)
	if options.OutputForce != "" {
		slogs.Infof("Force output directory: %s", options.OutputForce)
	}
	slogs.Infof("Download Proxy: %s", options.ProxyURL)
	slogs.Infof("Connect timeout: %ds", options.ConnectTimeout)
	slogs.Infof("Attempt timeout (legacy): %ds", options.IdleTimeout)
	slogs.Infof("Read idle timeout: %ds (0: disabled)", options.ReadIdleTimeout)
	slogs.Infof("Maximum attempts per URL: %d", options.Retries)
	slogs.Infof("Max concurrent: %d", options.MaxConcurrent)
	if options.MaxSpeed > 0 {
		slogs.Infof("Max speed: %d bytes/s", options.MaxSpeed)
	} else {
		slogs.Infof("Max speed: unlimited")
	}
	slogs.Infof("Force update: %v", options.UpdateForce)
	slogs.Infof("Enable force: %v", options.EnableForce)
}
