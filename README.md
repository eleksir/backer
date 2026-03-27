# Backer

HTTP/HTTPS backup server that creates compressed archives on-the-fly from configured directories. Supports multiple compression algorithms: gzip, pgzip, bzip2, zstd, lz4, xz.

## How to build it

```bash
make                                    # Build binary
make build VERSION=0.1.0                # Build with version
make test                               # Run tests
make                                    # Clean + build
make upgrade                            # Update dependencies
```

That will produce binary named `backer`. Copy it somewhere in `/usr/local/sbin/backer`, for example. In `data`
directory resides `config_example.json`. Using this file make working config, put it into `/etc/backer.json`.

Service ready to run.

You can copy systemd unit from contrib directory to /etc/systemd/system and make usual rituals required for systemd.

If you running alpine linux or gentoo, you can use openrc init script from contrib dir.

If you running FreeBSD, use the rc.d script `contrib/backer.freebsd` and place it in `/usr/local/etc/rc.d/`.

## How to use it

Build, install and configure backer. On your backup storage server you can do

```bash
curl -u username:password https://your_server:8086/archive -o "backup-$(date +%Y%m%d-%H%M%S).tar.gz"
```

The filename and extension depend on your `filename_prefix` and `compression_algorithm` settings.

Of course, something can go wrong, so you must check http status code after downloading backup.

## About config options

All options described in example config.

| Option | Default value | Wtf | Notes |
| :--- | :---- | :--- | :--- |
| address | "0.0.0.0" | Address at which server listens. | In many cases 0.0.0.0 is okay. |
| port | 8086 | Port at which server binds. | Number picked after intel first commercially successful cpu. |
| cert | — | Path to ssl certificate file. | Required when nohttps is false. |
| key | — | Path to ssl certificates key file. | Required when nohttps is false. |
| nohttps | false | Disable https and use plain http. | For lab experiments or development. |
| location | "/archive" | API endpoint path for backup download. | |
| user | — | Username for basic auth. | Required. |
| password | — | Password for basic auth. | Required. |
| log | stderr | Log output file path. | |
| loglevel | "info" | Verbosity of logs. | Options: error, warn, info, debug. |
| directories | — | Directories to backup. | Required. Array of paths. |
| backup_timeout | 60 | Timeout in minutes for backup streaming. | Range: 1-1440 minutes. |
| compression_level | 9 | Compression level. | 1 (fastest) to 9 (best compression). |
| exclude_patterns | [] | Regex patterns to exclude from backup. | E.g., `[".*\\.tmp$", "/node_modules/"]`. |
| filename_prefix | "backup" | Prefix for backup filename. | E.g., "mybackup" produces `mybackup-20260325-092341.tar.gz`. |
| compression_algorithm | "gzip" | Compression algorithm. | Options: gzip, pgzip, bzip2, zstd, lz4, xz. |

## Compression algorithms

| Algorithm | Speed | Compression | Notes |
| :--- | :--- | :--- | :--- |
| gzip | Fast | Good | Default. Standard, widely compatible. |
| pgzip | Fast | Good | Parallel gzip. Uses multiple CPU cores, slightly larger files. |
| bzip2 | Slow | Better | Pure Go. Good compression, slower than gzip. |
| zstd | Fast | Very good | Good balance of speed and compression. |
| lz4 | Fastest | Moderate | Extremely fast, larger files. |
| xz | Slowest | Best | Best compression, slowest speed. |

## Best practices

Traditional yada-yada-yada about internet threats.

It is entirely possible to use this utility without https/certificate. But you should not do this, really. Validity
of certificate is up to you, it can be even expired, but it should be well secured with reasonable key length and
good enough cypher.

And one more thing user and password are mandatory, we should have at least some minimal barriers, right?

## Special Thanks

This project was developed with the assistance of **opencode** (AI co-programmer powered by MIMO). The AI helped with:
- Code implementation and refactoring
- Test coverage
- Documentation
- Bug fixes
- Linting and code style consistency

Big thanks to the AI for doing most of the boring stuff while I was drinking coffee.
