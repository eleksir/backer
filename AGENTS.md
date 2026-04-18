# AGENTS.md

## Project Overview

Backer is a Go-based HTTP/HTTPS backup server that creates compressed archives on-the-fly from configured directories. Supports multiple compression algorithms: gzip, pgzip, bzip2, zstd, lz4, xz. It serves a single endpoint that clients (e.g., `curl`) can query to download a timestamped backup archive. Licensed under MIT.

## Fundamental Rules

- **Do not apply changes without explicit user consent.** When asked to analyze, review, or suggest any changes, present your proposal first and wait for confirmation before editing any files. Always ask "Should I implement this?" or "Should I apply these changes?" before modifying anything.
- **Keep codebase simple, small, maintainable, safe and human-readable.** Favour minimal, clear solutions over clever or abstract ones. Prefer clarity over cleverness. Prioritize explicitness. Minimize magic. Do not introduce patterns, layers, or indirection unless they directly solve a concrete problem. Every line of code must earn its place.
- **Create small focused modules and separate functionality over files.** Do not make universal implementation; try to move functions to separate files and name files after function names.
- **Use a consistent structure.** Subsequent searches will be faster and consume less context.
- **Avoid duplicate names.** This is not strict rule. If codebase contain multiple files with the same name (e.g., index.ts), we have to read them all to discover the one exactly we need. This consumes time and context.
- **When stuck, ask clarification questions, or present a short plan, or both.** Do not try to guess intent or make assumptions - ask.
- **Always make TODO plan before any job and follow it closely.**
- **Report any found defects, issues, bugs.** And keep track of them.
- **Comment the code.** Add code comments to explain corner cases, complex logic, business rules, and design decisions. Focus on **what** problem the code is solving rather than **how** it is solving it. Write comments in easy to read and understand language, for humans to comprehend. Every public API and structure field must have a docstring.
- **Use domain names.** Use naming conventions that reflect their purpose and domain concepts, not their technical implementation (e.g., OrderProcessor instead of OrderServiceFactory).
- **Test corner cases.** Write comprehensive test suites that cover various scenarios and edge cases.
- **Use tags and labels.** In comments, add tags or labels to files, classes, or functions to categorize them based on functionality, domain concepts, or design patterns.
- **Avoid obvious comments that do not add value.**
- **Remove dead code.** A half-finished refactoring can actually harm the contributions.
- **Update AGENTS.md automatically.** Keep AGENTS.md dense, detailed and concise.

## Workflow Rules

- **Don't use `context` package unless absolutely necessary.** Try to find another solution first.
- **Create functional and unit tests if possible/necessary** to verify changes that software code works at is should.

### Test Rules

- **Test data must be self-contained and created at test setup.** All test data should be created in `TestMain` in `internal/backer/main_test.go` before any tests run, not rely on pre-existing files in the repository. Use `test_data/test1/` as the test data directory.

- **Use `test_data/tmp/` for temporary files.** When tests need temporary storage, use this directory (not system `/tmp`) to keep test data isolated and predictable.

- **Always check errors with meaningful diagnostics.** Never discard errors with `_ =`. Use patterns like:
  ```go
  if err := os.MkdirAll(dir, 0755); err != nil {
      t.Fatalf("Failed to create directory %s: %v", dir, err)
  }
  ```

- **Tests must not depend on external state.** Test data should be created fresh for each test run and cleaned up afterward. Use `TestMain` with setup/cleanup functions.

- **Use `t.Fatal` or `t.Fatalf` for setup errors.** When test setup fails, fail fast with clear diagnostic messages showing what failed and why.

- **Reset global state between tests.** For tests that modify global variables (like `C` config), save the original value and restore it in a `defer`:
  ```go
  original := C
  defer func() { C = original }()
  ```

- **Skip tests gracefully when features are unavailable.** For platform-specific features (hardlinks, symlinks), use `t.Skip()` when the feature is not supported:
  ```go
  if err := os.Link(orig, link); err != nil {
      t.Skip("Hard links not supported, skipping test")
  }
  ```

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

### Design Decisions (Not Issues)

The following are intentional design decisions, not bugs or defects:

- **No rate limiting**: Backups are assumed to be infrequent enough that rate limiting is unnecessary.
- **Backup scheduling**: Client-side decision — server serves backups on-request only.
- **Basic Auth only**: HTTP Basic Auth is sufficient for a backup server behind a firewall.
- **Global config**: The `C Config` global variable is intentional — config is loaded once at startup.
- **No incremental backup**: Each backup is a full snapshot, not incremental.
- **Credentials in config file**: Config file should have restricted permissions (documented).
- **Thread-safe logging**: The global `Log` variable uses mutex protection to enable parallel test execution; this is not needed for single-server production use but allows tests to run in parallel without race conditions.

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
| dir_scan_timeout | `60` | Directory scan timeout in minutes (1-1440) |
| compression_level | `9` | Gzip level (1-9) |
| exclude_patterns | `[]` | Regex patterns to exclude from backup |
| filename_prefix | `backup` | Prefix for backup filename in Content-Disposition header |
| default_compression | `gzip` | Default compression algorithm: gzip, pgzip, bzip2, zstd, lz4 or xz |
| compression_algorithm | `gzip` | Alias for default_compression (deprecated) |

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
- `copyWithContext(ctx, dst, src)` — buffered copy (32KB) with cancellation checks, computes MD5 hash of copied data

### Special File Type Handling

- **Regular files**: streamed into tar archive with content
- **Directories**: included as tar directory entries
- **Symlinks**: stored as symlink entries (target path preserved). Backer never resolves or follows symlinks during directory traversal — it uses `os.Lstat` and `filepath.WalkDir` which do not follow symlinks. This means symlink cycles are impossible to occur.
- **Device files** (char/block): included with correct major/minor
- **Named pipes**: included (header only, no data)
- **Sockets**: skipped (not archivable)
- **Hard links**: deduplicated — if multiple files share the same inode, only the first is stored with content, subsequent hard links are stored as link entries pointing to the original

### Pipe Error Recovery

`isPipeClosedError()` checks for `"io: read/write on closed pipe"` errors to avoid noisy logging when a client disconnects mid-stream.

### Auth Handler

HTTP Basic Auth. Both username AND password must match exactly. Returns `401 Unauthorized` with `WWW-Authenticate` header on failure. Uses timing-safe comparison to prevent timing attacks. The condition is intentionally written without De Morgan's law simplification for clarity (linter suppression `//nolint:staticcheck`).

### Client IP Detection

Checks `X-Forwarded-For` header first (for proxied requests), then falls back to `RemoteAddr`.

### Server Configuration

- `ReadHeaderTimeout`: 5 seconds
- `WriteTimeout`: configurable via `backup_timeout` (converted to `time.Duration`)
- `IdleTimeout`: 10 seconds (for keep-alive connections; backer disables keep-alives explicitly)
- `MaxHeaderBytes`: 1KB (prevents header-based DoS attacks)
- `DisableKeepAlives`: true (each connection closed after request, no connection reuse)
- TLS minimum version: TLS 1.3 (when HTTPS enabled)

### TLS Error Handling

The server uses a `ServerWrapper` type to intercept and classify errors from `Serve()` and `ServeTLS()`. Two mechanisms:

1. **Server errors** (`Serve()` / `ServeTLS()`): Uses `errors.As` to detect TLS error types (`tls.AlertError`, `tls.CertificateVerificationError`, `tls.ECHRejectionError`, `tls.RecordHeaderError`) and checks for "tls:" prefix. TLS-specific errors logged at debug level, others at warn level.

2. **net/http internal errors** (`http.Server.ErrorLog`): Uses `DebugLogger()` which routes messages through `slogWriter.Write()`:
   - TLS-related errors (keywords: "tls", "ssl", "handshake", "certificate") → logged at debug level
   - Other errors → logged at warn level
   - Both are filtered by configured `loglevel`. Set `loglevel: debug` to see all messages.

This ensures client-side TLS errors (handshake failures, certificate issues) don't clutter production logs by default.

### Compression Level Mapping

The `compression_level` config option (1-9) is mapped differently for each compression algorithm:

| Algorithm | Level Mapping |
|-----------|----------------|
| gzip | 1-9 directly maps to gzip compression levels |
| pgzip | 1-9 directly maps to pgzip compression levels |
| bzip2 | 1-9 directly maps to bzip2 compression levels |
| zstd | 1-3: Fastest, 4-6: Default, 7-9: BetterCompression |
| lz4 | 1-3: Fastest (0), 4-6: Default (1), 7-9: Best (2) |
| xz | Uses default compression (no configurable level) |

### HTTP Routing & Compression Selection

The server uses `http.NewServeMux()` for routing. Multiple routes are registered to support different compression formats:

```go
mux.Handle(C.Location, http.HandlerFunc(backupHandler))           // /archive
mux.Handle(C.Location+".tar.gz", http.HandlerFunc(backupHandler))  // /archive.tar.gz
mux.Handle(C.Location+".tar.xz", http.HandlerFunc(backupHandler))   // /archive.tar.xz
mux.Handle(C.Location+".tar.bz2", http.HandlerFunc(backupHandler)) // /archive.tar.bz2
// ... etc
```

The compression algorithm is determined by extracting the extension from the request path using `path.Ext()`:
- For `/archive.tar.xz`, `path.Ext()` returns `.tar.xz` → maps to xz
- For `/archive`, `path.Ext()` returns `""` → uses default compression

The `getCompressionAlgorithm()` function handles both full paths and bare extensions:
- Full path: `/archive.tar.xz` → `.tar.xz` → xz
- Bare extension: `tar.xz` → `.tar.xz` → xz
- Single ext: `xz` → `.xz` → xz

### Unmatched Route Logging

The server wraps the `http.ResponseWriter` to capture status codes and log unmatched routes:
- **404 Not Found**: Logged as `"Not Found from {clientIP} to {path}"`
- **405 Method Not Allowed**: Logged as `"Method Not Allowed from {clientIP} for {method} {path}"`

This uses a `responseWriterWrapper` that tracks the status code via the `WriteHeader()` method.

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

All test data is created by `TestMain` in `internal/backer/main_test.go` before tests run and cleaned up afterward. This ensures tests are independent of pre-existing files.

- `test_data/test1/foo/` — test directory with files, subdirs, empty_dir
- `test_data/test1/bar/` — test directory with files and a directory named `goodbye.txt/` (edge case: directory with file extension)
- `test_data/test1/hardlinks/` — test directory with hard-linked files (original.txt, hardlink1.txt, hardlink2.txt)
- `test_data/test1/symlinks/` — test directory with symbolic links
- `test_data/test1/empty_dir/` — empty directory for testing
- `test_data/tmp/` — reserved for test temporary files (use instead of system `/tmp`)

## Service Deployment

Init scripts provided in `contrib/`:
- **systemd** (`backer.service`): `ExecStart=/usr/local/sbin/backer`, restart on failure
- **OpenRC** (`backer`): uses `supervise-daemon`, creates log/pid dirs in `start_pre`
- **FreeBSD rc.d** (`backer.freebsd`): daemonizes via `/usr/sbin/daemon`
