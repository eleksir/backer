package backer

import (
	"time"
)

const (
	// readHeaderTimeout is the maximum duration for reading request headers.
	readHeaderTimeout = 5 * time.Second

	// copyBufferSize is the buffer size used for streaming file contents.
	copyBufferSize = 32 * 1024

	// portMin is the minimum valid port number.
	portMin = 1

	// portMax is the maximum valid port number.
	portMax = 65535

	// backupTimeoutMin is the minimum backup timeout in minutes.
	backupTimeoutMin = 1

	// backupTimeoutMax is the maximum backup timeout in minutes (24 hours).
	backupTimeoutMax = 1440

	// compressionLevelMin is the minimum compression level.
	compressionLevelMin = 1

	// compressionLevelMax is the maximum compression level.
	compressionLevelMax = 9

	// defaultPort is the default listen port.
	defaultPort = 8086

	// defaultBackupTimeout is the default backup timeout in minutes.
	defaultBackupTimeout = 60

	// defaultCompressionLevel is the default compression level.
	defaultCompressionLevel = 9

	// defaultLocation is the default API endpoint path.
	defaultLocation = "/archive"

	// defaultAddress is the default listen address.
	defaultAddress = "0.0.0.0"

	// defaultLogLevel is the default log verbosity.
	defaultLogLevel = "info"

	// defaultFilenamePrefix is the default prefix for backup filenames.
	defaultFilenamePrefix = "backup"

	// defaultDefaultCompression is the default compression algorithm.
	defaultDefaultCompression = "gzip"
)

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
