# downfile CLI

[简体中文](README.md) | English | [Library documentation](../../README.en.md)

A command-line tool for YAML-configured batch downloads, built on the downutils library. Requires Go 1.25.5 or later.

## Installation and usage

```bash
go install github.com/winezer0/downutils/cmd/downfile@latest
downfile --config config.yaml
downfile --config config.yaml --output ./downloads --concurrent 3 --no-bar
downfile --help
```

These commands use Unix-style options. On Windows, go-flags help uses `/option:value`, for example `downfile /config:config.yaml /output:downloads /concurrent:3 /no-bar`. Use `downfile /help` for help.

| Option | Default | Description |
|---|---|---|
| `-c, --config` | `config.yaml` | YAML configuration path |
| `-o, --output` | `downloads` | Override every item's `storage-dir`; pass `--output=""` to use item directories |
| `-u, --update` | `false` | Force updates |
| `-e, --enable` | `false` | Enable all items |
| `-q, --no-bar` | `false` | Hide progress bars |
| `--disable-resume` | `false` | Restart each attempt from the beginning |
| `--read-idle-timeout` | `0` | Response-body read idle timeout in seconds; 0 disables it; excludes throttling |
| `-p, --proxy` | Empty | HTTP/SOCKS5 proxy; otherwise use environment proxies |
| `-C, --concurrent` | `1` | Maximum concurrent downloads within each group |
| `-S, --max-speed` | `0` | Per-download bytes/second limit; 0 means unlimited |
| `-t, --connect-timeout` | `10` | Connection and response header timeout, in seconds |
| `-T, --idle-timeout` | `60` | Total timeout per download attempt, also used for HTTP idle connections, in seconds |
| `-r, --retries` | `3` | Total attempts per URL, including the first; use at least 1 |
| `--lf` | Empty | Log file path |
| `--ll` | `info` | Console log level |
| `--cf` | `M` | Console format: `text`, `json`, a mask such as `TLCM`, or `off` |
| `-v, --version` | `false` | Display the version |

The CLI defaults to the `downloads` output directory and a 10-second connection timeout. Library defaults use an empty output override and a 30-second connection timeout. Legacy timeout and total-attempt behavior remains compatible.

## Configuration

See the [library configuration reference](../../README.en.md#configuration) for YAML fields, update policies, checksums, and file size limits. Example files are available as [config.yaml](config.yaml) and [demo.yaml](demo.yaml). Relative configuration and output paths are resolved from the working directory, not the executable directory.

Only enabled items are downloaded unless `--enable` is set. Use `--output=""` to preserve each item’s `storage-dir`.

## Logging

```bash
downfile --config config.yaml --ll debug --lf download.log --cf json
```

`--ll` controls console verbosity; file output defaults to debug level and JSON format. Use `--cf off` to disable console logging. The CLI uses `github.com/winezer0/slogs`.

## Resume and results

Temporary `.download` files can be reused during retries within one batch. The batch entry point cleans these files before and after execution, so resume across CLI invocations is not guaranteed. Use a dedicated download directory; see [resume and temporary files](../../README.en.md#resume-and-temporary-files).

The CLI logs a summary and per-task results. Exit codes are 0 for success or all skipped, 1 for configuration errors or any failed task, and 130 for interruption. Error summaries go to stderr; slogs logging remains on stdout. Help and runtime messages use English. Progress uses ASCII characters on stderr and is disabled when redirected or TERM=dumb.

## Dependencies

`github.com/jessevdk/go-flags` parses command-line options; downloads use the [downutils library](../../README.en.md).
