# downfile CLI

简体中文 | [English](README.en.md) | [库文档](../../README.md)

基于 downutils 库的命令行下载工具，通过 YAML 配置执行批量下载。要求 Go 1.25.5 或更高版本。

Go 版本要求适用于源码安装或构建；运行预编译二进制不需要安装 Go。Windows、Linux、macOS 均配置了原生 CI。

## 安装与用法

```bash
go install github.com/winezer0/downutils/cmd/downfile@latest
downfile --config config.yaml
downfile --config config.yaml --output ./downloads --concurrent 3 --no-bar
downfile --help
```

上述命令使用 Unix 风格参数。Windows 下 go-flags 帮助显示 `/参数:值`，例如 `downfile /config:config.yaml /output:downloads /concurrent:3 /no-bar`，可用 `downfile /help` 查看帮助。

从源码目录直接运行也可使用 `go run ./cmd/downfile --help`（Windows 可用 `/help`）。`go install` 将程序安装到 GOBIN，未设置时通常在 GOPATH/bin，请将该目录加入 PATH。

Windows PowerShell 示例：

```powershell
.\downfile.exe /config:config.yaml /output:downloads /concurrent:3 /no-bar
.\downfile.exe /config:config.yaml /output: /disable-resume /read-idle-timeout:30
.\downfile.exe /version
```

`/output:` 表示空覆盖目录，使用各项 storage-dir。Linux/macOS 使用下载的产物时先赋予执行权限，例如：

```bash
chmod +x ./downfile-linux-amd64
./downfile-linux-amd64 --config config.yaml
```

macOS 或其他架构请使用对应产物文件名。

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-c, --config` | `config.yaml` | YAML 配置路径 |
| `-o, --output` | `downloads` | 覆盖所有任务的 `storage-dir`；传入 `--output=""` 可使用各任务目录 |
| `-u, --update` | `false` | 强制更新 |
| `-e, --enable` | `false` | 启用所有任务 |
| `-q, --no-bar` | `false` | 隐藏进度条 |
| `--disable-resume` | `false` | 每次尝试从头下载 |
| `--read-idle-timeout` | `0` | 等待响应体数据的空闲超时（秒），0 关闭；不计限速等待 |
| `-p, --proxy` | 空 | HTTP/SOCKS5 代理；未设置时使用环境代理 |
| `-C, --concurrent` | `1` | 每组最大并发数；0/1 为串行，不能为负数 |
| `-S, --max-speed` | `0` | 每个下载的速度上限（字节/秒），0 不限速 |
| `-t, --connect-timeout` | `10` | 连接及响应头超时（秒） |
| `-T, --idle-timeout` | `60` | 单次下载尝试的总超时，同时用于 HTTP 空闲连接超时（秒） |
| `-r, --retries` | `3` | 每个 URL 的总尝试次数，包含首次尝试；应至少为 1 |
| `--lf` | 空 | 日志文件路径 |
| `--ll` | `info` | 控制台日志级别 |
| `--cf` | `M` | 控制台格式：`text`、`json`、`TLCM` 等掩码或 `off` |
| `-v, --version` | `false` | 显示版本 |
| `-h, --help` | — | 显示帮助，不下载文件 |

CLI 的默认输出目录是 `downloads`、连接超时是 10 秒；库的默认输出目录为空、连接超时是 30 秒。原有超时和总尝试次数的参数行为保持兼容。

`--idle-timeout 0` 会使下载尝试立即超时，不表示关闭；大文件或限速下载需提高总超时。`--read-idle-timeout` 仅限制响应体读取等待，不包含响应头或限速等待，且不覆盖总超时。次数应至少为 1；超时和限速不能为负数。普通重试依次等待 2、4、6……秒，404 等错误可能直接切换下载源。

## 配置文件

YAML 字段、更新策略、校验和及大小限制见[库配置说明](../../README.md#配置说明)。本目录提供 [config.yaml](config.yaml) 和 [demo.yaml](demo.yaml) 示例。配置路径和输出路径中的相对路径基于当前工作目录，不是可执行文件所在目录。

默认只下载 `enable: true` 的任务；`--enable` 可启用所有任务。需要保留各任务的 `storage-dir` 时，传入 `--output=""`。

filename 应为不含目录分隔符的文件名；嵌套目录通过 storage-dir 设置。重复目标路径仅警告，不会阻止并发写入。配置组顺序不保证与 YAML 一致，各组逐组执行，并发限制在组内生效。

## 日志

```bash
downfile --config config.yaml --ll debug --lf download.log --cf json
```

`--ll` 控制控制台日志级别；文件默认记录 debug 及以上级别并使用 JSON 格式。`--cf off` 可关闭控制台日志。CLI 使用 `github.com/winezer0/slogs`。

关闭控制台日志不等于关闭进度，完全静默进度需同时设置 `--no-bar`。系统错误、文件名和服务端内容仍可能包含非英文文字。

## 续传与结果

同一批次重试可复用 `.download` 临时文件。批量入口在执行前后会清理这些文件，因此不保证跨次运行 CLI 时续传。建议使用专用下载目录，详情见[续传与临时文件](../../README.md#续传与临时文件)。

配置的大小和校验和检查在临时文件上完成，通过后才替换目标，失败保留旧目标。现有续传不验证 ETag/Last-Modified；可变 URL 应使用 `--disable-resume`。

CLI 的汇总和逐项结果由 slogs 记录到 stdout（受日志级别及格式开关影响），致命错误和失败摘要写入 stderr。帮助和程序提示为英文，版本输出写入 stdout。进度使用 ASCII 字符并写入 stderr；仅在 stderr 是终端且 TERM 不为 dumb 时显示，仅重定向 stdout 不会关闭进度。

| 退出码 | 含义 |
|---|---|
| 0 | 成功、全部跳过、没有启用项、帮助或版本输出 |
| 1 | 参数、配置、日志初始化错误或任一任务失败 |
| 130 | 下载期间捕获到中断或 SIGTERM 并正常结束取消流程 |

Ctrl+C 可取消请求、排队任务、限速及重试等待；Linux/macOS 也处理 SIGTERM。操作系统强制终止（例如 SIGKILL）不能保证返回上述退出码或完成清理。无参数时显示帮助并返回 0。

## 源码构建与发布

在仓库根目录运行：

```bash
go build ./cmd/downfile
go build -trimpath -ldflags "-X main.AppVersion=0.1.0 -X main.BuildDate=2026-10-28" ./cmd/downfile
```

默认生成本机平台的 downfile（Windows 为 downfile.exe）。第二条命令演示显式注入版本和日期，请替换为实际值；未注入时使用 options.go 的默认值，不自动推导 Git 版本或日期。build-win.bat 和 build-linux.bat 是 Windows 下的 amd64 构建辅助脚本，不会注入发布元数据。

[Native CI](../../.github/workflows/ci.yml) 在 Windows、Linux、macOS 原生 runner 上执行 vet、竞态测试和版本注入检查，从 go.mod 读取 Go 版本。它在 main 推送和 PR 时自动运行，也可手动触发；测试的是各 runner 的本机架构。

[Build and Release](../../.github/workflows/build_and_release.yml) 仅手动触发，输入不带 v 前缀的 version（如 `0.1.0`、`0.1.0-rc.1`）。通过三平台检查后，以 CGO_ENABLED=0 构建：

| 系统 | amd64 | arm64 |
|---|---|---|
| Windows | downfile-windows-amd64.exe | downfile-windows-arm64.exe |
| Linux | downfile-linux-amd64 | downfile-linux-arm64 |
| macOS | downfile-darwin-amd64 | downfile-darwin-arm64 |

默认 publish=false，只上传 Actions artifacts；显式设置 publish=true 才创建 `v<version>` 的 Release，并将带后缀的版本标记为 prerelease。普通推送不会发布。发布构建注入输入版本与 UTC 构建日期。

## 依赖

`github.com/jessevdk/go-flags` 负责命令行参数解析；下载逻辑由 [downutils 库](../../README.md) 提供。
