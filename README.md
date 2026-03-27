# Backer

Kind of backup utility. It makes tar.gz on-the-fly out of given in config.json files.

## How to build it

```bash
make                                    # Build binary
make build VERSION=0.1.0                # Build with version
make test                               # Run tests
backer                                  # Run
backer --version                        # Show version
backer -c /path/to/config.json          # Custom config
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

and you'll get your backup with timestamp in filename, like `backup-20260325-105100.tar.gz`

Of course, something can go wrong, so you must check http status code after downloading backup.

## About config options

AI suggests to fill this section with useful information, despite that all options described in example config.

| Option | Default value | Wtf | Notes |
| :--- | :---- | :--- | :--- |
| address | "0.0.0.0" | Address at which server listens. | In many cases 0.0.0.0 is okay. |
| port | 8086 | Port at which server binds | Number picked after intel first comercially successful cpu. |
| cert | "/path/to/ssl.crt" | Path to ssl certificate file | It is not enforced in obvious form, but you should limit access to certificate at least with file permissions. |
| key | "/path/to/ssl.key" | Path to ssl certificates key file | It is not enforced in obvious form, but you should limit access to certificates key at least with file permissions. |
| nohttps | false | ability to disable https and use plain http. | This option is for lab experiments or for development needs. In case of this utility you should consider using long enough certificate with well ciphers. |
| location | "/archive" | This is static location where curl (or more sophisticated client) should aim to download backup | /archive is okay, but you free to choose something more wild like "/backup", just for giggles. |
| user | | You should pick up some flashy username for basic auth to prevent unauthorized access to your backup. | Enough said. |
| password | | You should pick up something super-duper secretous to prevent unauthorized access to your backup. | Enough said. |
| log | stderr | Where log messages are directed. If no value set, logs drops to stderr | |
| loglevel | | Verbosity of logs. | Can be error, warn, info, debug. Pick whatever you like. |
| directories | | An array with directories. Backer can work only with directories, not single files. | Put here all the folders you'd like to backup! |
| backup_timeout | 60 | Timeout in minutes for backup streaming operations. | If a backup takes longer than this, the connection will be terminated. Useful for large backups. Range: 1-1440 minutes (1 day max). |
| compression_level | 9 | Gzip compression level for tar.gz archives. | 1 (fastest) to 9 (best compression). Higher values produce smaller files but take longer to compress. |
| exclude_patterns | [] | Regex patterns to exclude from backup. | E.g., `[".*\\.tmp$", "/node_modules/"]`. Files matching any pattern are skipped. |
| filename_prefix | "backup" | Prefix for backup filename in Content-Disposition header. | E.g., "mybackup" produces `mybackup-20260325-092341.tar.gz`. |
| compression_algorithm | "gzip" | Compression algorithm for archive. | Options: "gzip", "bzip2", "zstd". Gzip is fastest, bzip2 compresses better, zstd is a good balance. |

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
