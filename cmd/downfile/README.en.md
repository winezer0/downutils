# downfile CLI

[简体中文](README.md) | English | [Library documentation](../../README.en.md)

A command-line tool for YAML-configured batch downloads, built on the downutils library. Requires Go 1.25.5 or later.

The Go requirement applies to source installation/builds; prebuilt binaries do not require Go. Native CI is configured for Windows, Linux, and macOS.

## Installation and usage

```bash
go install github.com/winezer0/downutils/cmd/downfile@latest
downfile --config config.yaml
downfile --config config.yaml --output ./downloads --concurrent 3 --no-bar
downfile --help
```

These commands use Unix-style options. On Windows, go-flags help uses `/option:value`, for example `downfile /config:config.yaml /output:downloads /concurrent:3 /no-bar`. Use `downfile /help` for help.

From the repository root, use `go run ./cmd/downfile --help` (or `/help` on Windows). `go install` places the executable in GOBIN, usually GOPATH/bin when GOBIN is unset; add that directory to PATH.

Windows PowerShell examples:

```powershell
.\downfile.exe /config:config.yaml /output:downloads /concurrent:3 /no-bar
.\downfile.exe /config:config.yaml /output: /disable-resume /read-idle-timeout:30
.\downfile.exe /version
```

`/output:` supplies an empty directory override to preserve item storage-dir values. For downloaded Linux/macOS binaries, grant executable permission first, for example:

```bash
chmod +x ./downfile-linux-amd64
./downfile-linux-amd64 --config config.yaml
```

Use the corresponding artifact name for macOS or another architecture.

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
| `-C, --concurrent` | `1` | Concurrency per group; 0/1 means serial; must not be negative |
| `-S, --max-speed` | `0` | Per-download bytes/second limit; 0 means unlimited |
| `-t, --connect-timeout` | `10` | Connection and response header timeout, in seconds |
| `-T, --idle-timeout` | `60` | Total timeout per download attempt, also used for HTTP idle connections, in seconds |
| `-r, --retries` | `3` | Total attempts per URL, including the first; use at least 1 |
| `--lf` | Empty | Log file path |
| `--ll` | `info` | Console log level |
| `--cf` | `M` | Console format: `text`, `json`, a mask such as `TLCM`, or `off` |
| `-v, --version` | `false` | Display the version |
| `-h, --help` | — | Display help without downloading |

The CLI defaults to the `downloads` output directory and a 10-second connection timeout. Library defaults use an empty output override and a 30-second connection timeout. Legacy timeout and total-attempt behavior remains compatible.

`--idle-timeout 0` expires attempts immediately rather than disabling the timeout. Increase the total timeout for large or throttled downloads. `--read-idle-timeout` limits response-body read waits, excludes headers and throttling, and does not override the total timeout. Attempts must be at least 1; timeouts and speed must not be negative. Ordinary retries wait 2, 4, 6, ... seconds; errors such as 404 can switch sources early.

## Configuration

See the [library configuration reference](../../README.en.md#configuration) for YAML fields, update policies, checksums, and file size limits. Example files are available as [config.yaml](config.yaml) and [demo.yaml](demo.yaml). Relative configuration and output paths are resolved from the working directory, not the executable directory.

Only enabled items are downloaded unless `--enable` is set. Use `--output=""` to preserve each item’s `storage-dir`.

Use a plain filename without directory separators; specify nested directories through storage-dir. Duplicate destinations only produce warnings and do not prevent concurrent writes. Group order is not guaranteed to match YAML order. Groups run sequentially, with concurrency limited within each group.

## Logging

```bash
downfile --config config.yaml --ll debug --lf download.log --cf json
```

`--ll` controls console verbosity; file output defaults to debug level and JSON format. Use `--cf off` to disable console logging. The CLI uses `github.com/winezer0/slogs`.

Disabling console logging does not disable progress; also set `--no-bar` to suppress it. System errors, filenames, and server content can still contain non-English text.

## Resume and results

Temporary `.download` files can be reused during retries within one batch. The batch entry point cleans these files before and after execution, so resume across CLI invocations is not guaranteed. Use a dedicated download directory; see [resume and temporary files](../../README.en.md#resume-and-temporary-files).

Configured size and checksum checks run against temporary files before replacing the destination; verification failure preserves the old destination. Resume does not validate ETag/Last-Modified; use `--disable-resume` for mutable URLs.

Summaries and per-task results use slogs on stdout, subject to logging level/format settings. Fatal errors and failure summaries go to stderr. Help and program messages use English; version output uses stdout. ASCII progress uses stderr only when it is a terminal and TERM is not dumb; redirecting stdout alone does not disable progress.

| Exit code | Meaning |
|---|---|
| 0 | Success, all skipped, no enabled items, help, or version output |
| 1 | Argument, configuration, logging initialization error, or any failed task |
| 130 | An interruption or SIGTERM was caught during downloads and cancellation completed normally |

Ctrl+C cancels requests, queued tasks, throttling, and retry waits. Linux/macOS also handle SIGTERM. Forced OS termination such as SIGKILL cannot guarantee these exit codes or completed cleanup. With no arguments, the CLI prints help and exits 0.

## Source builds and releases

From the repository root:

```bash
go build ./cmd/downfile
go build -trimpath -ldflags "-X main.AppVersion=0.1.0 -X main.BuildDate=2026-10-28" ./cmd/downfile
```

The output is downfile for the host platform, or downfile.exe on Windows. The second command demonstrates metadata injection; replace the version/date with actual values. Without injection, options.go defaults are used rather than automatic Git version/date discovery. build-win.bat and build-linux.bat are Windows-hosted amd64 build helpers and do not inject release metadata.

[Native CI](../../.github/workflows/ci.yml) uses native Windows, Linux, and macOS runners for vet, race tests, and metadata injection checks, reading the Go version from go.mod. It runs on main pushes and pull requests and supports manual dispatch. It tests each runner's native architecture.

[Build and Release](../../.github/workflows/build_and_release.yml) is manual only. Supply version without a v prefix, such as `0.1.0` or `0.1.0-rc.1`. After three-platform checks pass, it builds with CGO_ENABLED=0:

| OS | amd64 | arm64 |
|---|---|---|
| Windows | downfile-windows-amd64.exe | downfile-windows-arm64.exe |
| Linux | downfile-linux-amd64 | downfile-linux-arm64 |
| macOS | downfile-darwin-amd64 | downfile-darwin-arm64 |

The default publish=false only uploads Actions artifacts. Explicit publish=true creates a Release tagged `v<version>` and marks suffixed versions as prerelease. Ordinary pushes do not publish. Release builds inject the input version and UTC build date.

## Dependencies

`github.com/jessevdk/go-flags` parses command-line options; downloads use the [downutils library](../../README.en.md).
