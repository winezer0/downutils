# downutils

一个功能强大的 Golang 文件下载工具包，支持基于 YAML 配置的批量下载、断点续传、文件校验、并发下载等特性。

## 功能特点

- **多源下载**：支持为每个文件配置多个下载 URL，自动尝试直到成功
- **断点续传**：基于 URL 哈希的临时文件管理，支持中断后自动续传
- **文件校验**：支持 MD5/SHA1/SHA256 校验和文件大小验证
- **并发下载**：可配置最大并发数，支持串行或并行下载
- **进度显示**：实时显示下载进度，文件大小未知时自动切换为 spinner
- **下载限速**：基于令牌桶算法的速率控制
- **智能更新**：灵活的更新策略，支持按小时/天数控制更新频率
- **代理支持**：支持 HTTP 和 SOCKS5 代理
- **自动清理**：下载前后自动清理未完成的临时文件
- **错误分类**：详细的错误类型分类，便于问题排查
- **GitHub 支持**：自动转换 GitHub blob URL 为原始内容 URL

## 安装

```bash
go get github.com/your-org/downutils
```

## 快速开始

### 1. 加载配置文件

```yaml
# config.yaml
databases:
  - module: qqwry
    filename: qqwry.dat
    download-urls:
      - https://github.com/example/qqwry.dat/releases/latest/download/qqwry.dat
    keep-updated: "24h"
    storage-dir: ./data
    enable: true
    checksum: "md5:abc123def456"
    check-size: "1MB"
```

### 2. 执行下载

```go
package main

import (
    "github.com/your-org/downutils"
)

func main() {
    // 加载配置
    config, err := downutils.LoadConfig("config.yaml")
    if err != nil {
        panic(err)
    }

    // 使用默认选项执行下载
    options := downutils.NewDefaultDownOptions()
    result, err := downutils.ExecuteDownloads(options, config)
    if err != nil {
        panic(err)
    }

    // 显示下载结果
    downutils.DisplayDownloadResult(result)
}
```

## 配置说明

### DownConfig 结构

配置文件采用 YAML 格式，结构为 `map[string][]DownItem`：

```yaml
group_name:  # 配置组名称，用于日志输出
  - module: string        # 任务模块名称（必填）
    filename: string      # 保存的文件名（必填）
    download-urls:        # 下载 URL 列表（必填，支持多个源）
      - https://url1.com/file
      - https://url2.com/file
    keep-updated: string  # 更新策略（可选）
    storage-dir: string   # 存储目录（可选）
    enable: bool          # 是否启用（可选，默认 false）
    checksum: string      # 文件校验值（可选）
    check-size: string    # 文件大小校验（可选）
```

### 更新策略 (keep-updated)

| 值 | 说明 |
|---|---|
| `disable` / `no` / `false` | 文件存在时不更新 |
| `enable` / `yes` / `true` | 总是更新 |
| `1h` / `24h` / `72h` | 文件修改时间超过指定小时后更新 |
| `1d` / `7d` / `30d` | 文件修改时间超过指定天数后更新 |

### 文件校验 (checksum)

格式：`算法:校验值`，算法为空时默认使用 MD5

```yaml
checksum: "md5:abc123"        # MD5 校验
checksum: "sha1:abc123"       # SHA1 校验
checksum: "sha256:abc123"     # SHA256 校验
checksum: "abc123"            # 默认使用 MD5
```

### 文件大小校验 (check-size)

格式：`数字+单位`，支持 KB/MB/GB

```yaml
check-size: "1KB"    # 1 KB
check-size: "1.5MB"  # 1.5 MB
check-size: "2GB"    # 2 GB
check-size: "1024"   # 1024 字节
```

## API 文档

### 核心函数

#### ExecuteDownloads

执行完整的下载流程。

```go
func ExecuteDownloads(options *DownOptions, downConfig DownConfig) (*DownloadResult, error)
```

**参数：**
- `options`: 下载配置选项
- `downConfig`: 已加载的下载配置

**返回：**
- `DownloadResult`: 下载结果统计
- `error`: 错误信息

#### ExecuteDownloadsWithClient

使用已有的 HTTP 客户端执行下载。

```go
func ExecuteDownloadsWithClient(options *DownOptions, downConfig DownConfig, httpClient *http.Client) (*DownloadResult, error)
```

适用于需要复用 HTTP 客户端的场景。

#### LoadConfig

加载 YAML 配置文件。

```go
func LoadConfig(filename string) (DownConfig, error)
```

### 配置选项

#### DownOptions

```go
type DownOptions struct {
    OutputForce    string // 强制保存目录（优先级最高）
    UpdateForce    bool   // 强制更新（忽略 KeepUpdated 策略）
    EnableForce    bool   // 强制启用（忽略 Enable 字段）
    ShowProgress   bool   // 显示进度条（默认 true）
    MaxRetries     int    // 最大重试次数（默认 3）
    MaxConcurrent  int    // 最大并发数（默认 1，串行下载）
    MaxSpeed       int64  // 最大下载速度（字节/秒，0 表示不限速）
    ConnectTimeout int    // 连接超时（秒，默认 30）
    IdleTimeout    int    // 空闲超时（秒，默认 60）
    ProxyURL       string // 代理 URL
}
```

#### 获取默认配置

```go
options := downutils.NewDefaultDownOptions()
```

### HTTP 客户端

#### CreateHTTPClient

创建 HTTP 客户端。

```go
func CreateHTTPClient(config *HttpClientConfig) (*http.Client, error)
```

#### HttpClientConfig

```go
type HttpClientConfig struct {
    ConnectTimeout int    // 连接超时（秒）
    IdleTimeout    int    // 空闲超时（秒）
    ProxyURL       string // 代理 URL（支持 http 和 socks5）
}
```

### 结果统计

#### DownloadResult

```go
type DownloadResult struct {
    TotalItems   int        // 总下载项数
    SuccessItems int        // 成功下载项数
    FailedItems  int        // 失败下载项数
    SkippedItems int        // 跳过下载项数
    TaskInfos    []TaskInfo // 每个任务的详细信息
}
```

#### TaskInfo

```go
type TaskInfo struct {
    Module      string     // 任务模块名称
    FileName    string     // 文件名
    StoragePath string     // 完整存储路径
    DownloadURL string     // 实际使用的下载 URL
    Status      TaskStatus // 任务状态
    Resumed     bool       // 是否断点续传
    ErrorMsg    string     // 错误原因（成功时为空）
}
```

#### TaskStatus

```go
const (
    TaskStatusSuccess   TaskStatus = "success"    // 下载成功
    TaskStatusSkipped   TaskStatus = "skipped"    // 跳过（不需要更新）
    TaskStatusFailed    TaskStatus = "failed"     // 下载失败
    TaskStatusConfigErr TaskStatus = "config_err" // 配置错误
)
```

## 高级用法

### 自定义 HTTP 客户端

```go
// 创建自定义 HTTP 客户端
clientConfig := &downutils.HttpClientConfig{
    ConnectTimeout: 60,
    IdleTimeout:    120,
    ProxyURL:       "http://127.0.0.1:8080",
}

httpClient, err := downutils.CreateHTTPClient(clientConfig)
if err != nil {
    panic(err)
}
defer httpClient.CloseIdleConnections()

// 使用自定义客户端执行下载
result, err := downutils.ExecuteDownloadsWithClient(options, config, httpClient)
```

### 并发下载

```go
options := downutils.NewDefaultDownOptions()
options.MaxConcurrent = 5 // 最多 5 个并发下载

result, err := downutils.ExecuteDownloads(options, config)
```

### 下载限速

```go
options := downutils.NewDefaultDownOptions()
options.MaxSpeed = 1024 * 1024 // 限制为 1MB/s

result, err := downutils.ExecuteDownloads(options, config)
```

### 强制更新所有文件

```go
options := downutils.NewDefaultDownOptions()
options.UpdateForce = true // 忽略 keep-updated 策略

result, err := downutils.ExecuteDownloads(options, config)
```

### 下载所有项（包括 enable=false）

```go
options := downutils.NewDefaultDownOptions()
options.EnableForce = true // 忽略 enable 字段

result, err := downutils.ExecuteDownloads(options, config)
```

### 统一输出目录

```go
options := downutils.NewDefaultDownOptions()
options.OutputForce = "./downloads" // 所有文件保存到此目录

result, err := downutils.ExecuteDownloads(options, config)
```

## 错误处理

downutils 定义了详细的错误类型：

```go
const (
    ErrResourceNotFound   = "RESOURCE_NOT_FOUND"   // 404 资源不存在
    ErrDownloadCancelled  = "DOWNLOAD_CANCELLED"   // 下载被取消
    ErrNetworkError       = "NETWORK_ERROR"        // 网络错误
    ErrDiskFull           = "DISK_FULL"            // 磁盘空间不足
    ErrPermissionDenied   = "PERMISSION_DENIED"    // 权限不足
    ErrTimeout            = "TIMEOUT"              // 超时
    ErrChecksumMismatch   = "CHECKSUM_MISMATCH"    // 校验失败
    ErrCreateTempFile     = "CREATE_TEMP_FILE_FAILED" // 创建临时文件失败
)
```

## 依赖

- `gopkg.in/yaml.v3` - YAML 解析
- `github.com/schollz/progressbar/v3` - 进度条显示
- `github.com/winezer0/xutils/logging` - 日志记录
- `github.com/winezer0/xutils/progress` - 进度条封装

## 许可证

MIT License
