# downfile CLI

简体中文 | [English](README.en.md) | [库文档](../../README.md)

基于 downutils 库的命令行下载工具，通过 YAML 配置执行批量下载。要求 Go 1.25.5 或更高版本。

## 安装与用法

```bash
go install github.com/winezer0/downutils/cmd/downfile@latest
downfile --config config.yaml
downfile --config config.yaml --output ./downloads --concurrent 3 --no-bar
downfile --help
```

上述命令使用 Unix 风格参数。Windows 下 go-flags 帮助显示 `/参数:值`，例如 `downfile /config:config.yaml /output:downloads /concurrent:3 /no-bar`，可用 `downfile /help` 查看帮助。

| 参数 | 默认值 | 说明 |
|---|---|---|
| `-c, --config` | `config.yaml` | YAML 配置路径 |
| `-o, --output` | `downloads` | 覆盖所有任务的 `storage-dir`；传入 `--output=""` 可使用各任务目录 |
| `-u, --update` | `false` | 强制更新 |
| `-e, --enable` | `false` | 启用所有任务 |
| `-q, --no-bar` | `false` | 隐藏进度条 |
| `-p, --proxy` | 空 | HTTP/SOCKS5 代理；未设置时使用环境代理 |
| `-C, --concurrent` | `1` | 每个配置组的最大并发下载数 |
| `-S, --max-speed` | `0` | 每个下载的速度上限（字节/秒），0 不限速 |
| `-t, --connect-timeout` | `10` | 连接及响应头超时（秒） |
| `-T, --idle-timeout` | `60` | 单次下载尝试的总超时，同时用于 HTTP 空闲连接超时（秒） |
| `-r, --retries` | `3` | 每个 URL 的总尝试次数，包含首次尝试；应至少为 1 |
| `--lf` | 空 | 日志文件路径 |
| `--ll` | `info` | 控制台日志级别 |
| `--cf` | `M` | 控制台格式：`text`、`json`、`TLCM` 等掩码或 `off` |
| `-v, --version` | `false` | 显示版本 |

CLI 的默认输出目录是 `downloads`、连接超时是 10 秒；库的默认输出目录为空、连接超时是 30 秒。CLI 尚未提供 `DisableResume` 参数。

## 配置文件

YAML 字段、更新策略、校验和及大小限制见[库配置说明](../../README.md#配置说明)。本目录提供 [config.yaml](config.yaml) 和 [demo.yaml](demo.yaml) 示例。配置路径和输出路径中的相对路径基于当前工作目录，不是可执行文件所在目录。

默认只下载 `enable: true` 的任务；`--enable` 可启用所有任务。需要保留各任务的 `storage-dir` 时，传入 `--output=""`。

## 日志

```bash
downfile --config config.yaml --ll debug --lf download.log --cf json
```

`--ll` 控制控制台日志级别；文件默认记录 debug 及以上级别并使用 JSON 格式。`--cf off` 可关闭控制台日志。CLI 使用 `github.com/winezer0/slogs`。

## 续传与结果

同一批次重试可复用 `.download` 临时文件。批量入口在执行前后会清理这些文件，因此不保证跨次运行 CLI 时续传。建议使用专用下载目录，详情见[续传与临时文件](../../README.md#续传与临时文件)。

CLI 会记录汇总和逐项结果。当前单项任务失败不会强制返回非零退出码，应检查结果日志，而不能只依赖进程退出码判断全部任务是否成功。

## 依赖

`github.com/jessevdk/go-flags` 负责命令行参数解析；下载逻辑由 [downutils 库](../../README.md) 提供。
