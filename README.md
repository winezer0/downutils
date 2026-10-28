# downutils

简体中文 | [English](README.en.md)

[![Native CI](https://github.com/winezer0/downutils/actions/workflows/ci.yml/badge.svg)](https://github.com/winezer0/downutils/actions/workflows/ci.yml)

[命令行工具文档](cmd/downfile/README.md)

Go 文件下载库，支持 YAML 批量配置、同批次重试续传、文件校验和并发下载，另提供 downfile 命令行工具。

## 功能特点

下载库使用 `github.com/winezer0/slogs` 的默认日志实例。宿主可通过 `slogs.Init` 或 `slogs.SetDefault` 配置日志。

- **多源下载**：支持为每个文件配置多个下载 URL，自动尝试直到成功
- **断点续传**：基于 URL 哈希的临时文件管理，支持同一批次重试时续传
- **文件校验**：支持 MD5/SHA1/SHA256 校验和文件大小验证
- **安全替换**：临时文件通过配置的校验后才替换目标，校验失败保留原目标文件
- **取消与超时**：Context 取消可传递到请求、调度、限速和重试等待；支持独立响应体读取空闲超时
- **并发下载**：可配置最大并发数，支持串行或并行下载
- **进度显示**：stderr 为终端且 TERM 不为 dumb 时显示 ASCII 进度；大小未知时使用 spinner
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
        panic("some downloads failed; inspect TaskInfos")
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

相对路径基于进程工作目录，不是配置文件所在目录。批量入口要求 `filename` 是不含目录分隔符的文件名；各平台统一拒绝 `< > : " / \ | ? *`。Windows 保留名称未完整验证。重复目标路径只输出警告，不会阻止下载；请为不同任务配置不同目标路径。

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

空值关闭校验；算法名不区分大小写。未知算法会警告并按 MD5 处理，算法后没有校验值则关闭校验；当前配置验证不检查摘要长度或十六进制格式。

### 文件大小校验 (check-size)

格式：`数字+单位`，支持 KB/MB/GB（按 1024 换算），表示精确文件大小；最大大小限制请使用 `max-size`。

```yaml
check-size: "1KB"    # 1 KB
check-size: "1.5MB"  # 1.5 MB
check-size: "2GB"    # 2 GB
check-size: "1024"   # 1024 字节
```

也支持 B/K/M/G，单位不区分大小写。空值、0 或无法解析的数值关闭校验；未知单位警告后按字节处理，并不会导致配置验证失败。`max-size` 为整数硬上限，覆盖没有 Content-Length 的响应。

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

后台服务可使用带上下文的入口，将取消信号传递到请求、下载、限速等待、任务调度和重试等待；nil Context 按 Background 处理：

```go
func ExecuteDownloadsContext(ctx context.Context, options *DownOptions, config DownConfig) (*DownloadResult, error)
func ExecuteDownloadsWithClientContext(ctx context.Context, options *DownOptions, config DownConfig, client *http.Client) (*DownloadResult, error)
```

单项下载失败或取消通过 `DownloadResult.TaskInfos` 与 `FailedItems` 返回，顶层 `error` 可能仍为 `nil`，调用者应检查任务结果。`DownItem.MaxSize`（YAML：`max-size`）可限制最大文件字节数，包含未提供 Content-Length 的响应。URL 内容可能变化时，设置 `DownOptions.DisableResume = true`，每次尝试重新下载，避免拼接不同版本的数据。

所有批量入口执行相同的配置和选项验证。自定义客户端必须非空，由调用方负责关闭空闲连接。

`options == nil` 使用默认选项。`ProcessDownItems(client, items, options)` 是较低层的单组入口：它验证选项，但不会执行批量入口的任务字段验证、enable 过滤和前后清理，也没有公开的 Context 变体；通常应优先使用上述批量入口。

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
    ShowProgress   bool   // 允许终端进度显示（默认 true）
    MaxRetries     int    // 每个 URL 的总尝试次数，含首次（默认 3，至少为 1）
    MaxConcurrent  int    // 每组最大并发数（默认 1，0/1 均为串行）
    MaxSpeed       int64  // 每个下载的最大速度（字节/秒，0 表示不限速）
    ConnectTimeout int    // 连接及响应头超时（秒，默认 30）
    IdleTimeout    int    // 单次尝试总超时及 HTTP 空闲连接超时（秒，默认 60）
    ReadIdleTimeout int   // 等待响应体数据的空闲超时（秒，默认 0 关闭；不计限速等待）
    ProxyURL       string // 代理 URL
}
```

#### 获取默认配置

```go
options := downutils.NewDefaultDownOptions()
```

建议获取默认选项后再修改字段；零值结构体不会自动填充默认值，且 MaxRetries=0 会被拒绝。配置组逐组处理，但组间顺序由 Go map 遍历决定，不保证 YAML 中的顺序；组内结果保留任务原顺序。并发数、限速和超时不能为负数。

### 超时和重试语义

- `MaxRetries` 沿用旧名称，表示每个 URL 的总尝试次数（含首次），不是额外重试次数。一般重试间隔为 2、4、6……秒；404 会跳至下一个源，部分其他错误也会提前结束当前源的重试。
- `IdleTimeout` 为每次尝试设置总期限，同时用于库创建客户端的空闲连接超时。**设为 0 会立即到期，不表示关闭下载超时。** 大文件或低速下载需要提高它。
- `ReadIdleTimeout` 默认 0（关闭），仅限制等待响应体读取的时间，数据到达后重新计时，不计限速等待；不代替连接、响应头或总超时。
- 上层 Context 的期限仍会约束下载。自定义客户端自身设置的 Timeout 也会生效。

```go
options := downutils.NewDefaultDownOptions()
options.IdleTimeout = 1800     // 每次尝试最多 30 分钟
options.ReadIdleTimeout = 30   // 读取响应体数据时最多等待 30 秒
result, err := downutils.ExecuteDownloadsContext(ctx, options, config)
```

此片段假定 `ctx` 和 `config` 已初始化；仍应检查 `err` 和任务结果。

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

`FailedItems` 包含 failed 和 config_err。`DownloadURL` 在成功时记录实际 URL；`Resumed` 仅在成功下载实际采用已有片段时为 true，服务器回退完整下载时为 false。目标已存在而跳过时，不会重新执行校验。

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

// 批量入口会自动验证配置和选项
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

`DownloadError` 包含 `StatusCode`、`Message` 和 `Type`。校验和或精确大小不符使用 ErrChecksumMismatch，超过 max-size 使用 ErrMaximumSizeExceeded。错误类型常量包括：

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

错误常量用于内部分类，不代表所有错误都会转换为 DownloadError。磁盘满分类使用各平台原生错误，并支持 errors.Is 穿透包装：Windows 识别 ERROR_DISK_FULL / ERROR_HANDLE_DISK_FULL，Linux/macOS 识别 ENOSPC / EDQUOT（配额耗尽）。批量结果保存状态及错误文字，不保留可供 errors.As 使用的原始 error。

## 续传与临时文件

临时文件名为 `<目标路径>.<URL哈希>.download`。同一批次内的后续尝试可复用临时文件；服务器不支持 Range 时回退为完整下载。

现有续传不校验 ETag/Last-Modified；内容可能变化的 URL 应设置 `DisableResume=true`。大小及校验和检查在临时文件上执行，通过后替换目标文件：Windows 使用 MoveFileEx，Linux/macOS 使用 os.Rename。未配置校验时，完成传输后直接替换。

批量下载入口在执行前后会清理配置目录（或强制输出目录）中的 `.download` 文件，因此不保证跨批次或跨进程续传。清理也可能涉及目录中的其他同后缀文件，建议使用专用下载目录。

清理会递归扫描收集到的目录，没有保留期限或容量预算。未配置 storage-dir 且 OutputForce 为空时，默认当前目录不会被自动加入清理列表。

## 日志配置

库直接调用 `slogs.Debugf/Infof/Warnf/Errorf`，使用 slogs 默认实例，不使用标准库 `slog.Default()` 作为回退。宿主应在首次下载前配置日志：

```go
if err := slogs.Init(slogs.NewConfig("info", "download.log", "M")); err != nil {
    panic(err)
}
defer slogs.CloseAll() // 在所有使用日志的任务结束后关闭
```

此片段需导入 `github.com/winezer0/slogs`。`Init` 在默认实例已初始化后不会重新配置；需要替换时可通过 `slogs.NewLogger` 和 `slogs.SetDefault` 设置。`NewConfig` 的级别参数控制控制台，文件默认记录 debug 及以上级别并使用 JSON 格式；控制台传入 `off` 可关闭控制台输出。

## 平台支持与验证

支持 Windows、Linux 和 macOS。文件路径使用 filepath 处理，运行时提示为英文；系统返回的错误、路径和服务端内容仍可能含其他语言。

| 平台 | 发布构建目标 | 原生 CI |
|---|---|---|
| Windows | amd64、arm64 | windows-latest |
| Linux | amd64、arm64 | ubuntu-latest |
| macOS（darwin） | amd64、arm64 | macos-latest |

原生 CI 在每种 runner 的本机架构执行，不代表六种系统与架构组合均完成实机测试。发布产物以 CGO_ENABLED=0 交叉编译；本地使用库或源码构建需 Go 1.25.5 或更高版本，运行预编译 CLI 无需安装 Go。Linux/macOS 的二进制文件需要执行权限；本工具下载的其他程序不会自动 chmod。

## 开发、CI 与发布

```bash
go vet ./...
go test -race -count=1 -timeout=10m ./...
go build ./...
```

运行竞态检查需要支持的平台及 C 编译器。普通发布构建关闭 CGO，不需要 C 编译器。

- [Native CI](.github/workflows/ci.yml)：main 推送、PR、手动触发或其他工作流调用时运行三平台 vet、原生竞态测试、CLI 构建与版本元数据检查；Go 版本从 go.mod 读取。
- [Build and Release](.github/workflows/build_and_release.yml)：仅手动触发，要求不带 v 前缀的版本号，例如 `0.1.0` 或 `0.1.0-rc.1`。先通过三平台原生检查，再构建六种产物。
- `publish=false`（默认）仅上传 Actions artifacts，不创建 tag 或 Release。只有显式选择 `publish=true` 才发布 `v<版本号>`，带预发布后缀的版本标记为 prerelease。普通推送不会发布。
- 发布构建注入 `main.AppVersion` 和 `main.BuildDate`（UTC 日期）；源码直接构建未注入时使用 options.go 中的默认值，并非自动生成的版本与日期。

CLI 构建、产物名称及版本注入示例见[命令行文档](cmd/downfile/README.md#源码构建与发布)。

## 依赖

- `gopkg.in/yaml.v3` - YAML 解析
- `github.com/schollz/progressbar/v3` - 进度条显示
- `github.com/winezer0/slogs` - 日志记录
- `github.com/jessevdk/go-flags` - CLI 参数解析
- `golang.org/x/sys`、`golang.org/x/term` - 平台错误、文件替换及终端检测

## 许可证

原文标注为 MIT License；当前仓库未包含 LICENSE 文件，完整许可条款尚待补充。
