package downutils

import (
	"github.com/winezer0/xutils/logging"
	"os"
	"strconv"
	"strings"
	"time"
)

// DownConfig 配置文件结构
type DownConfig map[string][]DownItem

// DownItem 下载项目结构
type DownItem struct {
	Module       string   `yaml:"module" json:"module"`              // 自定义的任务模块名称
	FileName     string   `yaml:"filename" json:"fileName"`          // 自定义下载后的存储文件名
	DownloadURLs []string `yaml:"download-urls" json:"downloadUrls"` // 可选的下载URL列表,避免单个URl失效
	KeepUpdated  string   `yaml:"keep-updated" json:"keepUpdated"`   // 文件存在时的更新策略: disable/no/false-不更新, enable/yes/true-总是更新, 1h/24h/72h-指定小时后更新, 1d/7d-指定天后更新
	StorageDir   string   `yaml:"storage-dir" json:"storageDir"`     // 自定义存储目录，为空时使用默认目录
	Enable       bool     `yaml:"enable" json:"enable"`              // 是否开启任务 对于未开启的任务需要进行过滤
	Checksum     string   `yaml:"checksum" json:"checksum"`          // 文件校验值，格式：md5:xxx/sha1:xxx/sha256:xxx，前缀为空时默认使用md5算法
	CheckSize    string   `yaml:"check-size" json:"checkSize"`       // 文件大小校验，格式：1KB/1MB/1GB，为空或0时不校验文件大小
}

// DownOptions 下载配置选项，调用方传递额外的下载参数
type DownOptions struct {
	OutputForce    string // 强制保存目录：不为空时优先级最高，始终使用此目录保存文件
	UpdateForce    bool   // 强制更新：为 true 时忽略 KeepUpdated 策略，强制重新下载
	EnableForce    bool   // 强制启用：为 true 时忽略 Enable 字段，强制下载所有任务
	ShowProgress   bool   // 显示进度条：为 true 时显示下载进度，默认显示
	MaxRetries     int    // 下载错误 最大重试次数
	MaxConcurrent  int    // 最大并发下载数，默认为1(串行下载)
	MaxSpeed       int64  // 最大下载速度(字节/秒)，0表示不限速
	ConnectTimeout int    // HTTP 连接超时(秒)
	IdleTimeout    int    // HTTP 空闲超时(秒)
	ProxyURL       string // HTTP 代理URL
}

// HttpClientConfig HTTP客户端配置
type HttpClientConfig struct {
	ConnectTimeout int    // HTTP 连接超时时间(秒)
	IdleTimeout    int    // HTTP 空闲超时时间(秒)
	ProxyURL       string // HTTP 代理URL(支持http和socks5)
}

// DownloadTaskConfig 单次下载任务配置
type DownloadTaskConfig struct {
	Checksum     string // 文件校验值
	CheckSize    string // 文件大小校验
	ShowProgress bool   // 是否显示进度条
	MaxSpeed     int64  // 最大下载速度(字节/秒)
}

// DownloadError 自定义下载错误类型
type DownloadError struct {
	StatusCode int
	Message    string
	Type       string
}

func (e DownloadError) Error() string {
	return e.Message
}

// NewDefaultDownOptions 返回默认的下载配置选项
func NewDefaultDownOptions() *DownOptions {
	return &DownOptions{
		OutputForce:    "",
		UpdateForce:    false,
		EnableForce:    false,
		ShowProgress:   true,
		MaxRetries:     3,
		MaxConcurrent:  1,
		MaxSpeed:       0,
		ConnectTimeout: 30,
		IdleTimeout:    60,
		ProxyURL:       "",
	}
}

// DefaultClientConfig 返回默认的HTTP客户端配置
func DefaultClientConfig() *HttpClientConfig {
	return &HttpClientConfig{
		ConnectTimeout: 30, // 默认30秒连接超时
		IdleTimeout:    60, // 默认60秒空闲超时
		ProxyURL:       "", // 默认不使用代理
	}
}

// 常量定义
const (
	// DownloadBufferSize 下载缓冲区大小
	DownloadBufferSize = 32 * 1024 // 32KB
)

// 错误类型常量
const (
	// ErrResourceNotFound 资源不存在错误(404)
	ErrResourceNotFound = "RESOURCE_NOT_FOUND"
	// ErrDownloadCancelled 下载被取消错误
	ErrDownloadCancelled = "DOWNLOAD_CANCELLED"
	// ErrNetworkError 网络错误
	ErrNetworkError = "NETWORK_ERROR"
	// ErrDiskFull 磁盘空间不足
	ErrDiskFull = "DISK_FULL"
	// ErrPermissionDenied 权限不足
	ErrPermissionDenied = "PERMISSION_DENIED"
	// ErrTimeout 超时错误
	ErrTimeout = "TIMEOUT"
	// ErrChecksumMismatch 文件校验值或大小不匹配
	ErrChecksumMismatch = "CHECKSUM_MISMATCH"
	// ErrCreateTempFile 创建临时文件失败
	ErrCreateTempFile = "CREATE_TEMP_FILE_FAILED"
)

// ParseKeepUpdated 解析更新策略，返回是否需要更新
// keepUpdated 格式：disable/no/false-不更新, enable/yes/true-总是更新, 1h/24h/72h-指定小时后更新, 1d/7d-指定天后更新
// fileModTime 文件修改时间，如果文件不存在则为零值
// 当输入无法识别的关键字时，返回 false(不更新)并输出警告日志
func ParseKeepUpdated(keepUpdated string, fileModTime time.Time) bool {
	// 空字符串默认为 false(不更新)
	if keepUpdated == "" {
		return false
	}

	// 转为小写进行比较
	strategy := strings.ToLower(strings.TrimSpace(keepUpdated))

	// disable/no/false: 不更新
	if strategy == "disable" || strategy == "no" || strategy == "false" {
		return false
	}

	// enable/yes/true: 总是更新
	if strategy == "enable" || strategy == "yes" || strategy == "true" {
		return true
	}

	// 解析小时格式：1h, 24h, 72h 等
	if strings.HasSuffix(strategy, "h") {
		hourStr := strings.TrimSuffix(strategy, "h")
		hours, err := strconv.Atoi(hourStr)
		if err != nil || hours <= 0 {
			logging.Warnf("invalid hours value '%s' in keep-updated strategy, treating as disable", keepUpdated)
			return false
		}

		// 如果文件不存在(零值时间)，需要下载
		if fileModTime.IsZero() {
			return true
		}

		// 检查文件修改时间是否超过指定小时数
		expireTime := fileModTime.Add(time.Duration(hours) * time.Hour)
		return time.Now().After(expireTime)
	}

	// 解析天数格式：1d, 7d, 30d 等
	if strings.HasSuffix(strategy, "d") {
		dayStr := strings.TrimSuffix(strategy, "d")
		days, err := strconv.Atoi(dayStr)
		if err != nil || days <= 0 {
			logging.Warnf("invalid days value '%s' in keep-updated strategy, treating as disable", keepUpdated)
			return false
		}

		// 如果文件不存在(零值时间)，需要下载
		if fileModTime.IsZero() {
			return true
		}

		// 检查文件修改时间是否超过指定天数
		expireTime := fileModTime.Add(time.Duration(days) * 24 * time.Hour)
		return time.Now().After(expireTime)
	}

	// 无法识别的策略，输出警告并返回 false
	logging.Warnf("unrecognized keep-updated strategy '%s', treating as disable", keepUpdated)
	return false
}

// GetFileModTime 获取文件修改时间，如果文件不存在返回零值
func GetFileModTime(filePath string) time.Time {
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
