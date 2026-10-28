# downutils

[简体中文](README.md) | English

[CLI documentation](cmd/downfile/README.en.md)

A Go file download library with YAML batch configuration, multiple download sources, retries, resumable downloads, checksums, concurrency, progress indicators, and rate limits.

## Features

- Multiple URLs per file, tried in order until a download succeeds.
- URL-hashed temporary files for resuming retries within a batch.
- MD5, SHA1, SHA256, exact file size verification, and maximum file size limits.
- Configurable concurrency within each configuration group.
- Progress bars, with a spinner when the file size is unknown.
- Per-download rate limits using a token bucket.
- Update policies based on file modification time, in hours or days.
- HTTP and SOCKS5 proxies, with environment proxy support when no explicit proxy is set.
- Cleanup of incomplete downloads before and after batch execution.
- Classified download errors and per-task results.
- Conversion of GitHub blob URLs to raw content URLs.

The library uses the default logger from `github.com/winezer0/slogs` directly. Configure it before the first download with `slogs.Init` or `slogs.SetDefault`.

## Installation

The current `go.mod` requires Go 1.25.5 or later. Library source files are at the repository root.

```bash
go get github.com/winezer0/downutils
```

## Quick start

### 1. Create a configuration file

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
    # Set checksum and check-size for the actual file, or omit them.
    max-size: 104857600 # Optional: maximum 100 MiB
```

The URL is a placeholder. Replace it with a real download URL before running the example.

### 2. Run the downloads

```go
package main

import "github.com/winezer0/downutils"

func main() {
    config, err := downutils.LoadConfig("config.yaml")
    if err != nil {
        panic(err)
    }

    options := downutils.NewDefaultDownOptions()
    result, err := downutils.ExecuteDownloads(options, config)
    if err != nil {
        panic(err)
    }

    downutils.DisplayDownloadResult(result)
    if result.FailedItems > 0 {
        panic("some downloads failed; inspect TaskInfos")
    }
}
```

## Configuration

### DownConfig

The YAML structure is `map[string][]DownItem`:

```yaml
group_name:              # Group name used in logs
  - module: string       # Required task name
    filename: string     # Required destination filename
    download-urls:       # Required list of download sources
      - https://url1.com/file
      - https://url2.com/file
    keep-updated: string # Optional update policy
    storage-dir: string  # Optional destination directory
    enable: bool         # Optional; defaults to false
    checksum: string     # Optional checksum
    check-size: string   # Optional exact file size
    max-size: int64      # Optional maximum bytes; 0 disables the limit; must not be negative
```

Destination directory priority is `DownOptions.OutputForce` > `storage-dir` > the current directory. Disabled items are excluded from results unless `EnableForce` is set.

### Update policy (`keep-updated`)

| Value | Behavior when the destination file exists |
|---|---|
| `disable` / `no` / `false` | Keep the existing file |
| `enable` / `yes` / `true` | Always update |
| `1h` / `24h` / `72h` | Update after the specified number of hours since file modification |
| `1d` / `7d` / `30d` | Update after the specified number of days since file modification |

Empty or unrecognized values disable updates. Missing destination files are still downloaded.

### Checksums (`checksum`)

Use `algorithm:value`. Without a prefix, MD5 is used. The values below are placeholders for actual checksums.

```yaml
checksum: "md5:abc123"
checksum: "sha1:abc123"
checksum: "sha256:abc123"
checksum: "abc123" # MD5 by default
```

### File size (`check-size`)

Use a number followed by an optional unit. KB/MB/GB use powers of 1024. This verifies the exact file size; use `max-size` for an upper bound.

```yaml
check-size: "1KB"
check-size: "1.5MB"
check-size: "2GB"
check-size: "1024" # Bytes
```

`max-size` is an integer byte limit, including for responses without Content-Length. Zero disables the limit.

## API

The signatures below describe exported functions in package `downutils`.

### Batch downloads

```go
func ExecuteDownloads(options *DownOptions, config DownConfig) (*DownloadResult, error)
func ExecuteDownloadsWithClient(options *DownOptions, config DownConfig, client *http.Client) (*DownloadResult, error)
func ExecuteDownloadsContext(ctx context.Context, options *DownOptions, config DownConfig) (*DownloadResult, error)
func ExecuteDownloadsWithClientContext(ctx context.Context, options *DownOptions, config DownConfig, client *http.Client) (*DownloadResult, error)
```

Use the context variants to propagate cancellation to HTTP requests, transfers, and retry waits. A nil options pointer uses library defaults.

Individual download failures and cancellations are reported in `DownloadResult.TaskInfos` and `FailedItems`; the returned top-level error can still be nil. Always inspect task results.

`ExecuteDownloads` and `ExecuteDownloadsContext` validate the configuration. The variants accepting a custom client do not perform the same validation: call `ValidateDownConfig(config, options.EnableForce)` beforehand and pass a non-nil client. The caller owns the custom client's idle connection cleanup.

### Configuration loading and defaults

```go
func LoadConfig(filename string) (DownConfig, error)
func ValidateDownConfig(config DownConfig, enableForce bool) []string
func NewDefaultDownOptions() *DownOptions
func DefaultClientConfig() *HttpClientConfig
```

`LoadConfig` reads and parses YAML; validation is a separate step. `ValidateDownConfig` returns validation messages and checks enabled items, or all items when `enableForce` is true.

### DownOptions

```go
type DownOptions struct {
    DisableResume  bool   // Restart each attempt; useful for mutable URLs
    OutputForce    string // Override destination directories; default empty
    UpdateForce    bool   // Ignore update policies
    EnableForce    bool   // Include disabled items
    ShowProgress   bool   // Default true
    MaxRetries     int    // Total attempts per URL, including the first; default 3, at least 1
    MaxConcurrent  int    // Concurrency within each group; default 1
    MaxSpeed       int64  // Per-download bytes/second; 0 means unlimited
    ConnectTimeout int    // Connection and response header timeout; default 30 seconds
    IdleTimeout    int    // Total attempt timeout and HTTP idle connection timeout; default 60 seconds
    ProxyURL       string // Explicit proxy URL; empty uses environment proxies
}
```

Use `NewDefaultDownOptions()` and override the fields you need. A zero-valued struct does not provide these defaults. Groups are processed sequentially. `MaxSpeed` is a per-download limit, not a shared limit across concurrent downloads.

### HTTP clients

```go
func CreateHTTPClient(config *HttpClientConfig) (*http.Client, error)

type HttpClientConfig struct {
    ConnectTimeout int    // Connection and response header timeout, in seconds
    IdleTimeout    int    // HTTP idle connection timeout, in seconds
    ProxyURL       string // HTTP/SOCKS5 proxy; empty uses environment proxies
}
```

Passing nil to `CreateHTTPClient` uses `DefaultClientConfig()` (30-second connection and 60-second idle connection timeouts). The client has no overall timeout; batch execution applies the per-attempt timeout from `DownOptions.IdleTimeout`.

### Results

```go
type DownloadResult struct {
    TotalItems   int
    SuccessItems int
    FailedItems  int
    SkippedItems int
    TaskInfos    []TaskInfo
}

type TaskInfo struct {
    Module      string
    FileName    string
    StoragePath string
    DownloadURL string
    Status      TaskStatus
    Resumed     bool
    ErrorMsg    string
}

const (
    TaskStatusSuccess   TaskStatus = "success"
    TaskStatusSkipped   TaskStatus = "skipped"
    TaskStatusFailed    TaskStatus = "failed"
    TaskStatusConfigErr TaskStatus = "config_err"
)
```

`TaskInfos` contains details for each processed item. `FailedItems` includes failed downloads and configuration error statuses. Use `DisplayDownloadResult(result)` to log a summary and per-task details.

## Advanced usage

### Reuse a custom HTTP client

```go
client, err := downutils.CreateHTTPClient(&downutils.HttpClientConfig{
    ConnectTimeout: 60,
    IdleTimeout:    120,
    ProxyURL:       "http://127.0.0.1:8080",
})
if err != nil {
    panic(err)
}
defer client.CloseIdleConnections()

if messages := downutils.ValidateDownConfig(config, options.EnableForce); len(messages) > 0 {
    panic(messages)
}
result, err := downutils.ExecuteDownloadsWithClient(options, config, client)
```

This fragment assumes `config` and `options` have already been initialized. Check both `err` and `result.FailedItems` as in the quick start.

### Concurrency, rate limits, and overrides

```go
options := downutils.NewDefaultDownOptions()
options.MaxConcurrent = 5
options.MaxSpeed = 1024 * 1024 // 1 MiB/s per download
options.UpdateForce = true   // Ignore keep-updated
options.EnableForce = true   // Include enable=false items
options.OutputForce = "./downloads"
options.DisableResume = true // Restart attempts for URLs whose content can change

result, err := downutils.ExecuteDownloads(options, config)
```

These options can be used independently. If URLs can serve different content between attempts, `DisableResume` avoids appending data from a different version.

## Errors

`DownloadError` contains `StatusCode`, `Message`, and `Type`. Error type constants include:

```go
const (
    ErrResourceNotFound    = "RESOURCE_NOT_FOUND"
    ErrDownloadCancelled   = "DOWNLOAD_CANCELLED"
    ErrNetworkError        = "NETWORK_ERROR"
    ErrDiskFull            = "DISK_FULL"
    ErrPermissionDenied    = "PERMISSION_DENIED"
    ErrTimeout             = "TIMEOUT"
    ErrChecksumMismatch    = "CHECKSUM_MISMATCH"
    ErrMaximumSizeExceeded = "MAXIMUM_SIZE_EXCEEDED"
    ErrCreateTempFile      = "CREATE_TEMP_FILE_FAILED"
)
```

Checksum and exact size failures use `ErrChecksumMismatch`; exceeding `max-size` uses `ErrMaximumSizeExceeded`. Batch results expose task failures through status and error message fields.

## Resume and temporary files

Temporary filenames follow `<destination>.<URL hash>.download`. Subsequent attempts within a batch can resume from the same temporary file. If the server does not support Range requests, the download restarts from the beginning.

Batch entry points clean `.download` files in configured directories (or the forced output directory) before and after execution. Resume across separate batches or processes is therefore not guaranteed. Cleanup may also remove other files with the same suffix in those directories; use dedicated download directories.

## Logging

The library calls `slogs.Debugf/Infof/Warnf/Errorf` directly using its default logger. It does not fall back to the standard library's `slog.Default()`. Initialize logging before the first download:

```go
if err := slogs.Init(slogs.NewConfig("info", "download.log", "M")); err != nil {
    panic(err)
}
defer slogs.CloseAll() // After all tasks using logging have finished
```

Import `github.com/winezer0/slogs` for this fragment. `Init` does not reconfigure an already initialized default logger. To replace it, create a logger with `slogs.NewLogger` and use `slogs.SetDefault`.

The level argument of `NewConfig` controls console output. File output defaults to debug level and JSON format. Use `off` as the console format to disable console output.

## Dependencies

- `gopkg.in/yaml.v3`: YAML parsing.
- `github.com/schollz/progressbar/v3`: Progress indicators.
- `github.com/winezer0/slogs`: Logging.

## License

The original README identifies the license as MIT. The repository currently has no LICENSE file; the full license terms still need to be added.
