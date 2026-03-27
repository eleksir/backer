package backer

type (
	// Config main config struct.
	Config struct {
		Address              string   `json:"address"`
		Port                 int      `json:"port"`
		Cert                 string   `json:"cert"`
		Key                  string   `json:"key"`
		NoHTTPS              bool     `json:"nohttps"`
		Location             string   `json:"location"`
		User                 string   `json:"user"`
		Password             string   `json:"password"`
		Log                  string   `json:"log"`
		LogLevel             string   `json:"loglevel"`
		Directories          []string `json:"directories"`
		BackupTimeout        int      `json:"backup_timeout"`        // Timeout in minutes for backup streaming.
		CompressionLevel     int      `json:"compression_level"`     // Gzip compression level (1-9, default 9).
		ExcludePatterns      []string `json:"exclude_patterns"`      // Regex patterns to exclude from backup.
		FilenamePrefix       string   `json:"filename_prefix"`       // Prefix for backup filename (default: "backup").
		CompressionAlgorithm string   `json:"compression_algorithm"` // Compression algorithm: gzip or bzip2 (default: "gzip").
	}
)

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
