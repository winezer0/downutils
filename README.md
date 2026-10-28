# downutils

简体中文 | [English](README.en.md)

[命令行工具文档](cmd/downfile/README.md)

一个功能强大的 Golang 文件下载工具包，支持基于 YAML 配置的批量下载、断点续传、文件校验、并发下载等特性。

## 功能特点

下载库使用 `github.com/winezer0/slogs` 的默认日志实例。宿主可通过 `slogs.Init` 或 `slogs.SetDefault` 配置日志。

- **多源下载**：支持为每个文件配置多个下载 URL，自动尝试直到成功
- **断点续传**：基于 URL 哈希的临时文件管理，支持同一批次重试时续传
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

当前 `go.mod` 要求 Go 1.25.5 或更高版本。库代码位于仓库根目录。

```bash
go get github.com/winezer0/downutils
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
    # 按实际文件填写 checksum 和 check-size，未知时省略
    max-size: 104857600 # 最大 100 MiB，可选
```

上面的 URL 为占位示例，使用前请替换为实际下载地址。

### 2. 执行下载

```go
package main

import (
    "github.com/winezer0/downutils"
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
    if result.FailedItems > 0 {
        panic("部分下载任务失败，请检查 TaskInfos")
    }
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
    max-size: int64       # 最大文件字节数（可选，0 不限制，不能为负数）
```

保存目录优先级为 `DownOptions.OutputForce` > `storage-dir` > 当前目录。未启用的任务不会计入结果，除非设置 `EnableForce`。

### 更新策略 (keep-updated)

| 值 | 说明 |
|---|---|
| `disable` / `no` / `false` | 文件存在时不更新 |
| `enable` / `yes` / `true` | 总是更新 |
| `1h` / `24h` / `72h` | 文件修改时间超过指定小时后更新 |
| `1d` / `7d` / `30d` | 文件修改时间超过指定天数后更新 |

空值或无法识别的值按不更新处理；目标文件不存在时仍会下载。

### 文件校验 (checksum)

格式：`算法:校验值`，算法为空时默认使用 MD5

下面的校验值仅为格式示例，使用时须替换为实际文件的完整校验值。

```yaml
checksum: "md5:abc123"        # MD5 校验
checksum: "sha1:abc123"       # SHA1 校验
checksum: "sha256:abc123"     # SHA256 校验
checksum: "abc123"            # 默认使用 MD5
```

### 文件大小校验 (check-size)

格式：`数字+单位`，支持 KB/MB/GB（按 1024 换算），表示精确文件大小；最大大小限制请使用 `max-size`。

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

后台服务可使用带上下文的入口，将取消信号传递到请求、下载和重试等待：

```go
func ExecuteDownloadsContext(ctx context.Context, options *DownOptions, config DownConfig) (*DownloadResult, error)
func ExecuteDownloadsWithClientContext(ctx context.Context, options *DownOptions, config DownConfig, client *http.Client) (*DownloadResult, error)
```

单项下载失败或取消通过 `DownloadResult.TaskInfos` 与 `FailedItems` 返回，顶层 `error` 可能仍为 `nil`，调用者应检查任务结果。`DownItem.MaxSize`（YAML：`max-size`）可限制最大文件字节数，包含未提供 Content-Length 的响应。URL 内容可能变化时，设置 `DownOptions.DisableResume = true`，每次尝试重新下载，避免拼接不同版本的数据。

`ExecuteDownloads` / `ExecuteDownloadsContext` 会验证配置。使用自定义客户端的两个入口不会执行同样的配置验证，调用前可使用 `ValidateDownConfig(config, options.EnableForce)` 检查，并传入非空客户端；客户端由调用方负责关闭空闲连接。

#### LoadConfig

加载 YAML 配置文件。

此函数只读取并解析 YAML，不验证任务字段；可使用 `ValidateDownConfig(config, enableForce)` 获取验证错误列表。

```go
func LoadConfig(filename string) (DownConfig, error)
```

### 配置选项

#### DownOptions

```go
type DownOptions struct {
    DisableResume  bool   // 每次尝试重新下载，适用于内容可变化的 URL
    OutputForce    string // 强制保存目录（优先级最高）
    UpdateForce    bool   // 强制更新（忽略 KeepUpdated 策略）
    EnableForce    bool   // 强制启用（忽略 Enable 字段）
    ShowProgress   bool   // 显示进度条（默认 true）
    MaxRetries     int    // 每个 URL 的总尝试次数，含首次（默认 3，至少为 1）
    MaxConcurrent  int    // 最大并发数（默认 1，串行下载）
    MaxSpeed       int64  // 每个下载的最大速度（字节/秒，0 表示不限速）
    ConnectTimeout int    // 连接超时（秒，默认 30）
    IdleTimeout    int    // 单次尝试总超时及 HTTP 空闲连接超时（秒，默认 60）
    ProxyURL       string // 代理 URL
}
```

#### 获取默认配置

```go
options := downutils.NewDefaultDownOptions()
```

建议获取默认选项后再修改字段，零值结构体不会自动填充上述默认值。配置组按顺序逐组处理，并发数限制应用于每个组。

### HTTP 客户端

#### CreateHTTPClient

创建 HTTP 客户端。

```go
func CreateHTTPClient(config *HttpClientConfig) (*http.Client, error)
```

#### HttpClientConfig

```go
type HttpClientConfig struct {
    ConnectTimeout int    // 连接及响应头超时（秒）
    IdleTimeout    int    // HTTP 空闲连接超时（秒）
    ProxyURL       string // 代理 URL（支持 http 和 socks5）；空值使用环境代理
}
```

`CreateHTTPClient(nil)` 使用 `DefaultClientConfig()`：连接超时 30 秒、空闲连接超时 60 秒。客户端不设置整体超时；批量下载通过 `DownOptions.IdleTimeout` 设置每次尝试的总超时。

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
if messages := downutils.ValidateDownConfig(config, options.EnableForce); len(messages) > 0 {
    panic(messages)
}
result, err := downutils.ExecuteDownloadsWithClient(options, config, httpClient)
```

以上片段假定已经初始化 `options` 和 `config`，执行后仍需检查 `err` 和 `result.FailedItems`。

### 并发下载

```go
options := downutils.NewDefaultDownOptions()
options.MaxConcurrent = 5 // 最多 5 个并发下载

result, err := downutils.ExecuteDownloads(options, config)
```

### 下载限速

```go
options := downutils.NewDefaultDownOptions()
options.MaxSpeed = 1024 * 1024 // 每个下载限制为 1 MiB/s，并发时不是全局共享上限

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
    ErrMaximumSizeExceeded = "MAXIMUM_SIZE_EXCEEDED" // 超过最大文件大小
    ErrCreateTempFile     = "CREATE_TEMP_FILE_FAILED" // 创建临时文件失败
)
```

## 续传与临时文件

临时文件名为 `<目标路径>.<URL哈希>.download`。同一批次内的后续尝试可复用临时文件；服务器不支持 Range 时回退为完整下载。

批量下载入口在执行前后会清理配置目录（或强制输出目录）中的 `.download` 文件，因此不保证跨批次或跨进程续传。清理也可能涉及目录中的其他同后缀文件，建议使用专用下载目录。

## 日志配置

库直接调用 `slogs.Debugf/Infof/Warnf/Errorf`，使用 slogs 默认实例，不使用标准库 `slog.Default()` 作为回退。宿主应在首次下载前配置日志：

```go
if err := slogs.Init(slogs.NewConfig("info", "download.log", "M")); err != nil {
    panic(err)
}
defer slogs.CloseAll() // 在所有使用日志的任务结束后关闭
```

此片段需导入 `github.com/winezer0/slogs`。`Init` 在默认实例已初始化后不会重新配置；需要替换时可通过 `slogs.NewLogger` 和 `slogs.SetDefault` 设置。`NewConfig` 的级别参数控制控制台，文件默认记录 debug 及以上级别并使用 JSON 格式；控制台传入 `off` 可关闭控制台输出。

## 依赖

- `gopkg.in/yaml.v3` - YAML 解析
- `github.com/schollz/progressbar/v3` - 进度条显示
- `github.com/winezer0/slogs` - 日志记录

## 许可证

原文标注为 MIT License；当前仓库未包含 LICENSE 文件，完整许可条款尚待补充。
