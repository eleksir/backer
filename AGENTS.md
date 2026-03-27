# AGENTS.md

## Project Overview

Backer is a Go-based HTTP/HTTPS backup server that creates compressed archives on-the-fly from configured directories. Supports multiple compression algorithms: gzip, pgzip, bzip2, zstd, lz4, xz. It serves a single endpoint that clients (e.g., `curl`) can query to download a timestamped backup archive. Licensed under MIT.

## Workflow Rules

- **Do not apply changes without explicit user consent.** When asked to analyze, review, or suggest improvements, present your proposal first and wait for confirmation before editing any files. Always ask "Should I implement this?" or "Should I apply these changes?" before writing code.
- **Keep the codebase simple, small, maintainable, safe, and human-readable.** Favor minimal, clear solutions over clever or abstract ones. Do not introduce patterns, layers, or indirection unless they directly solve a concrete problem. Every line of code must earn its place.
- **When stuck, ask a clarification question, or propose a short plan, or both.** Do not guess intent or make assumptions — ask.
- **Always make a TODO plan before acting. And try to follow this plan.**
- **If you notice defects, bugs, or issues, always report it.**
- **Track new features, known issues, bugs, and other defects.** Update documentation as you implement features, fix issues, fix bugs, or discover new ones.
- **Update `AGENTS.md` as the project evolves.** Keep it comprehensive but dense.
- **Don't use `context` package unless absolutely necessary.** Try to find another solution first.
- **Create unit tests if possible/necessary** to verify changes that are not covered by unit tests.

## Build & Test Commands

```bash
make build                    # Build binary (version from git or "dev")
make build VERSION=0.1.0     # Build with specific version tag
make test                     # Run all tests (go test ./...)
make clean                    # Remove built binary
make                          # Clean + build
make upgrade                  # Update dependencies (go get -u, mod tidy, vendor)
make help                     # Show all targets
```

Version is injected at build time via ldflags: `-X main.version=$(VERSION)`.

## Linting

```bash
golangci-lint run             # Run linter (configured via .golangci.yml)
```

The project uses golangci-lint v2 with a strict set of linters. See `.golangci.yml` for the full configuration.

## Code Style & Conventions

- **Language**: Go 1.25
- **Formatting**: `gofmt` + `goimports` (enforced by linter)
- **Imports**: Grouped imports, single `import` block required (`grouper` linter)
- **Declaration order**: Type, const, var, func (`decorder` linter)
- **Named returns**: Avoid named return values, especially in `defer` blocks (`nonamedreturns` linter)
- **Naming**: Follow Go initialisms (HTTP, TLS, IP, etc.) as defined in `staticcheck` config
- **File footer**: Each source file ends with `/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */`

### Godoc Comments

All exported types, functions, and methods must have godoc comments (`godoclint`, `godot`). Comments must end with a period.

### Error Handling

- **Error messages must be capitalized** (e.g., `"Failed to read config"`, not `"failed to read config"`). The `ST1005` linter rule is disabled to allow this.
- Use `fmt.Errorf` with `%w` for error wrapping.
- Use `%v` for errors in format strings (not `%s`).
- Past tense style: `"Failed to..."`, `"Unable to..."`.
- Always handle errors explicitly; never discard with `_` except in tests where intentional.

### Logging Conventions

- Use `backer/internal/log` package (wrapper around Go's `slog`).
- Log levels: `error`, `warn`, `info`, `debug`.
- Past tense with context: `"Skipping %s: %v", filename, err`.
- Appropriate severity: Warn for recoverable issues, Error for problems, Fatal for startup failures.

### Config Validation Pattern

For each config option: set default first, then immediately validate range/required value:
```go
if C.Port == 0 {
    C.Port = 8086
    log.Warnf("Config option port is not set, fallback to %d", C.Port)
}

if C.Port < portMin || C.Port > portMax {
    return fmt.Errorf("Config option port must be between %d and %d, got %d", portMin, portMax, C.Port)
}
```

### Function Naming

```go
// Exported (public): VerbNoun pattern
NewServer()
LoadConfig()
CreateTarGzStream()
GetFilesFromDirectories()

// Unexported (private): verbNoun or isNoun pattern
writeWithContext()
copyWithContext()
isPipeClosedError()
defaultConfigPath()
```

### JSON Tags

Struct fields use PascalCase, json tags use snake_case:
```go
BackupTimeout     int `json:"backup_timeout"`
CompressionLevel  int `json:"compression_level"`
```

### Intentional Linter Suppressions

| Rule | Reason |
|------|--------|
| ST1005 | Disable — allows capitalized error messages (e.g., `"Failed to..."`) |
| QF1001 | De Morgan's law disabled for auth condition clarity |
| revive (error strings) | Same reason as ST1005 |

## Project Structure

```
cmd/
  backer/
    main.go                         # CLI entrypoint, flag parsing, TLS setup, server start

internal/
  backer/
    types.go                        # Config struct definition
    consts.go                       # Named constants for magic numbers
    globals.go                      # Global C Config variable, compiled excludePatterns
    archive.go                      # Common archive logic (pipe, tar, file iteration)
    gzip.go                         # CreateTarGzStream, CreateTarPgzipStream
    bzip2.go                        # CreateTarBzip2Stream
    zstd.go                         # CreateTarZstdStream, zstdLevel helper
    lz4.go                          # CreateTarLz4Stream
    xz.go                           # CreateTarXzStream
    loadConfig.go                   # HJSON/JSON config parsing with validation
    loadConfig_test.go              # Config validation tests
    newServer.go                    # HTTP server creation, auth handler, context helpers
    newServer_test.go               # Server, auth, and context helper tests
    getFilesFromDirectories.go      # Directory walking, exclusion filtering (isExcluded)
    getFilesFromDirectories_test.go # File traversal and exclude pattern tests
    utils_test.go                   # Archive stream tests
  log/
    main.go                         # Logging wrapper around stdlib slog
    main_test.go                    # Log package tests

data/
  config_example.json               # Annotated example config (HJSON with comments)

contrib/
  backer                            # OpenRC init script (Alpine/Gentoo)
  backer.service                    # systemd unit file
  backer.freebsd                    # FreeBSD rc.d script

test_data/
  test_config.json                  # Test config fixture
  example.crt / example.key         # TLS test certificates
  test1/                            # File/directory fixtures for tests

Makefile                            # Build targets
README.md                           # User documentation
LICENSE                             # MIT License
llms.txt                            # LLM-friendly project reference
```

## Configuration

Config is JSON (HJSON with comments supported). Default path depends on OS:
- Linux/macOS: `/etc/backer.json`
- FreeBSD/NetBSD/OpenBSD/DragonFly: `/usr/local/etc/backer.json`
- Override with `-c /path/to/config.json`

### Config Options

| Option | Default | Description |
|--------|---------|-------------|
| address | `0.0.0.0` | Listen address |
| port | `8086` | Listen port (1-65535) |
| cert | — | TLS certificate file path |
| key | — | TLS key file path |
| nohttps | `false` | Disable HTTPS (for dev/lab) |
| location | `/archive` | API endpoint path |
| user | — | Basic auth username (required) |
| password | — | Basic auth password (required) |
| log | stderr | Log output file path |
| loglevel | `info` | Log verbosity: error/warn/info/debug |
| directories | — | Directories to backup (required, validated for existence) |
| backup_timeout | `60` | Stream timeout in minutes (1-1440) |
| compression_level | `9` | Gzip level (1-9) |
| exclude_patterns | `[]` | Regex patterns to exclude from backup |
| filename_prefix | `backup` | Prefix for backup filename in Content-Disposition header |
| compression_algorithm | `gzip` | Compression algorithm: gzip, pgzip, bzip2, zstd, lz4 or xz |

## Architecture & Key Patterns

### Global Config

Uses a global `C Config` variable (`internal/backer/globals.go`). This is intentional — config is loaded once at startup and never modified concurrently.

### HJSON Config Parsing

Requires double conversion (hjson → map → json → struct):
```go
hjson.Unmarshal(buf, &tmp)  // parse HJSON into map[string]any
json.Marshal(tmp)            // re-encode as strict JSON
json.Unmarshal(buf, &C)     // decode into Config struct
```

### Streaming Archive Creation

No temp files on disk. Uses `io.Pipe` with a goroutine to stream archives on-the-fly:
- Common logic in `archive.go`: `createArchiveStream()` handles pipe, goroutine, tar writing, file iteration
- Each compression algorithm has its own file: `gzip.go`, `bzip2.go`, `zstd.go`, `lz4.go`, `xz.go`
- All return an `io.ReadCloser`
- HTTP handler copies from pipe reader directly to HTTP response

### Context-Aware I/O

Two helper functions in `newServer.go` handle context cancellation during streaming:
- `writeWithContext(ctx, fn)` — checks context before calling fn
- `copyWithContext(ctx, dst, src)` — buffered copy (32KB) with cancellation checks

### Special File Type Handling

- **Regular files**: streamed into tar archive with content
- **Directories**: included as tar directory entries
- **Symlinks**: stored as symlink entries (target path preserved)
- **Device files** (char/block): included with correct major/minor
- **Named pipes**: included (header only, no data)
- **Sockets**: skipped (not archivable)

### Pipe Error Recovery

`isPipeClosedError()` checks for `"io: read/write on closed pipe"` errors to avoid noisy logging when a client disconnects mid-stream.

### Auth Handler

HTTP Basic Auth. Both username AND password must match exactly. Returns `401 Unauthorized` with `WWW-Authenticate` header on failure. The condition is intentionally written without De Morgan's law simplification for clarity (linter suppression `//nolint:staticcheck`).

### Client IP Detection

Checks `X-Forwarded-For` header first (for proxied requests), then falls back to `RemoteAddr`.

### Server Configuration

- `ReadHeaderTimeout`: 5 seconds
- `WriteTimeout`: configurable via `backup_timeout` (converted to `time.Duration`)
- TLS minimum version: TLS 1.3 (when HTTPS enabled)

## Dependencies

- `github.com/hjson/hjson-go` v3.3.0 — HJSON config parsing
- `github.com/dsnet/compress` v0.0.1 — bzip2 compression
- `github.com/klauspost/compress` v1.18.5 — zstd compression
- `github.com/klauspost/pgzip` v1.2.6 — parallel gzip compression
- `github.com/pierrec/lz4` v2.6.1 — lz4 compression
- `github.com/ulikunitz/xz` v0.5.15 — xz compression

## Testing

```bash
go test ./...                    # Run all tests
go test -v -run TestName ./...   # Run specific test
make test                        # Via Makefile
```

Tests live alongside source files as `*_test.go` (package name, not `_test` suffix). Coverage is approximately 69% on `internal/backer` and ~100% on `internal/log`.

### Test Conventions

- Reset global config before tests: `C = Config{}`
- Reset exclude patterns: `excludePatterns = nil`
- Use `t.TempDir()` for temporary files
- Helper `writeTempConfig(t, content)` creates temp config files for validation tests
- Symlink tests skip gracefully with `t.Skip()` when not supported
- Context cancellation tests verify correct behavior

### Test Data

- `test_data/test_config.json` — valid test config (user: JohnnyGoode, password: SharpShooter)
- `test_data/test1/foo/` — test directory with files, subdirs, empty dirs
- `test_data/test1/bar/` — test directory with files and a directory named `goodbye.txt/` (edge case: directory with file extension)
- `test_data/example.crt` / `example.key` — TLS test certificates

## Service Deployment

Init scripts provided in `contrib/`:
- **systemd** (`backer.service`): `ExecStart=/usr/local/sbin/backer`, restart on failure
- **OpenRC** (`backer`): uses `supervise-daemon`, creates log/pid dirs in `start_pre`
- **FreeBSD rc.d** (`backer.freebsd`): daemonizes via `/usr/sbin/daemon`
