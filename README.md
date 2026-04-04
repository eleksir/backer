# Backer

HTTP/HTTPS backup server that creates compressed archives on-the-fly from configured directories. Supports multiple compression algorithms: gzip, pgzip (which is parallel gzip), bzip2, zstd, lz4, xz.

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
wget --quiet --no-check-certificate --content-disposition --http-user=username --http-password=password https://your_server:8086/archive
```

The filename and extension depend on your `filename_prefix` and `default_compression` settings.

You can also override compression by specifying the archive extension in the URL:

```bash
wget --quiet --no-check-certificate --content-disposition --http-user=username --http-password=password https://your_server:8086/archive.xz
wget --quiet --no-check-certificate --content-disposition --http-user=username --http-password=password https://your_server:8086/archive.gz    # Uses pgzip
wget --quiet --no-check-certificate --content-disposition --http-user=username --http-password=password https://your_server:8086/archive.bz2
wget --quiet --no-check-certificate --content-disposition --http-user=username --http-password=password https://your_server:8086/archive.zst
wget --quiet --no-check-certificate --content-disposition --http-user=username --http-password=password https://your_server:8086/archive.lz4
```

Note: `/archive.tar.gz` always uses pgzip (parallel gzip) for better performance.

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
| default_compression | "gzip" | Default compression algorithm. | Options: gzip, pgzip, bzip2, zstd, lz4, xz. |
| compression_algorithm | "gzip" | Alias for default_compression. | You should use default_compression instead. |

## Compression algorithms

AI suggests this table as expected performance.

| Algorithm | Speed | Compression | Notes |
| :--- | :--- | :--- | :--- |
| gzip | Fast | Good | Default. Standard, widely compatible. |
| pgzip | Fast | Good | Parallel gzip. Uses multiple CPU cores, slightly larger files. |
| bzip2 | Slow | Better | Pure Go. Good compression, slower than gzip. |
| zstd | Fast | Very good | Good balance of speed and compression. |
| lz4 | Fastest | Moderate | Extremely fast, larger files. |
| xz | Slowest | Best | Best compression, slowest speed. |

In real live with backup of 1 lxc container inside vm powered by 1 core cpu, 768Mb RAM and 100Mbit non-guaranteed
network bandwidth, located in near by country I've got these results

| Algorithm | Size       | Time    | CPU Load | RAM RSS+Shm, Mb | Notes |
| :---      | ---:       | ---:    | ---:     | ---:            | :---  |
| None      | 1461250551 |         |     0-2% |             0+0 | original dataset |
| lz4       | 1097745915 |  6m:12s |     4-6% |            30+6 | multithreaded lz4, pierrec/lz4 |
| zstd      |  614015773 |  4m:52s |   40-60% |            60+6 | klauspost/compress/zstd |
| pgzip     |  817163857 | 11m:25s |   96-97% |            37+6 | multithreaded (parallel gzip), fully gzip compatible, klauspost/pgzip |
| gzip      |  816734188 | 14m:30s |      96% |            18+6 | stdlib compression/gzip |
| bzip2     |  788061284 |  7m:16s |   92-94% |            43+6 | dsnet/compress/bzip2 |
| xz        |  512004612 | 13m:53s |      96% |           116+6 | ulikunitz/xz |

Obvously in case of lz4 and zstd the bottleneck was network. And I have to admit, that network definitely was not
100Mbit, but it was pretty hard to measure real bandwidth. Anyway let's pretend that during experiment it was roughly
the same.

## Best practices

Traditional yada-yada-yada about internet threats.

It is entirely possible to use this utility without https/certificate. But you should not do this, really. Validity
of certificate is up to you, it can be even expired, but it should be well secured with reasonable key length and
good enough cypher.

And one more thing user and password are mandatory, we should have at least some minimal barriers, right?

## Logging

Backer logs events via Go's `slog`. Log levels: error, warn, info, debug.

### TLS Error Handling

TLS handshake errors and other client-side SSL/TLS errors are logged at **debug** level to reduce log clutter.
This includes:
- Certificate verification failures
- Handshake failures
- Protocol errors
- TLS-specific connection issues

## Special Thanks

This project was developed with the assistance of **opencode** (AI co-programmer powered by MIMO). The AI helped with:
- Code implementation and refactoring
- Test coverage
- Documentation
- Bug fixes
- Linting and code style consistency

Big thanks to the AI for doing most of the boring stuff while I was drinking coffee.
