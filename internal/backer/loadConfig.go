package backer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"

	"backer/internal/log"

	hjson "github.com/hjson/hjson-go"
)

// LoadConfig loads config file to global variable C.
func LoadConfig(path string) error {
	var tmp map[string]any

	buf, err := os.ReadFile(path)

	if err != nil {
		return fmt.Errorf("Failed to read config file: %w", err)
	}

	if err = hjson.Unmarshal(buf, &tmp); err != nil {
		return fmt.Errorf("Unable to parse config: %w", err)
	}

	buf, err = json.Marshal(tmp)

	if err != nil {
		return fmt.Errorf("Unable to convert config: %w", err)
	}

	if err = json.Unmarshal(buf, &C); err != nil {
		return fmt.Errorf("Failed to decode config file: %w", err)
	}

	// Address: default and no range validation needed.
	if C.Address == "" {
		C.Address = defaultAddress

		log.Warnf("Config option address is not set, fallback to %s", C.Address)
	}

	// Port: set default, then validate range.
	if C.Port == 0 {
		C.Port = defaultPort

		log.Warnf("Config option port is not set, fallback to %d", C.Port)
	}

	if C.Port < portMin || C.Port > portMax {
		return fmt.Errorf("Config option port must be between %d and %d, got %d", portMin, portMax, C.Port)
	}

	// Location: default and no range validation needed.
	if C.Location == "" {
		C.Location = defaultLocation

		log.Warnf("Config option location is not set, fallback to %s", C.Location)
	}

	// Log: default only.
	if C.Log == "" {
		log.Warn("Config option log is not set, fallback to stderr")
	}

	// LogLevel: default and no range validation needed.
	if C.LogLevel == "" {
		C.LogLevel = defaultLogLevel

		log.Warnf("Config option loglevel is not set, fallback to %s", C.LogLevel)
	}

	// Directories: required, then validate existence.
	if len(C.Directories) == 0 {
		return errors.New("You should configure at least one directory to backup!") //nolint: revive
	}

	for _, dir := range C.Directories {
		if _, err := os.Stat(dir); err != nil {
			return fmt.Errorf("Backup directory does not exist: %s", dir)
		}
	}

	// BackupTimeout: set default, then validate range.
	if C.BackupTimeout == 0 {
		C.BackupTimeout = defaultBackupTimeout

		log.Warnf("Config option backup_timeout is not set, fallback to %d minutes", C.BackupTimeout)
	}

	if C.BackupTimeout < backupTimeoutMin || C.BackupTimeout > backupTimeoutMax {
		return fmt.Errorf("Config option backup_timeout must be between %d and %d minutes, got %d", backupTimeoutMin, backupTimeoutMax, C.BackupTimeout)
	}

	// CompressionLevel: set default, then validate range.
	if C.CompressionLevel == 0 {
		C.CompressionLevel = defaultCompressionLevel

		log.Warnf("Config option compression_level is not set, fallback to %d", C.CompressionLevel)
	}

	if C.CompressionLevel < compressionLevelMin || C.CompressionLevel > compressionLevelMax {
		return fmt.Errorf("Config option compression_level must be between %d and %d, got %d", compressionLevelMin, compressionLevelMax, C.CompressionLevel)
	}

	// HTTPS: validate cert/key when HTTPS enabled.
	if !C.NoHTTPS {
		switch {
		case C.Cert == "" && C.Key == "":
			return errors.New("Config option nohttps set to false, but neither cert nor key options set")
		case C.Cert == "":
			return errors.New("Config option nohttps set to false, but no cert option set")
		case C.Key == "":
			return errors.New("Config option nohttps set to false, but no key option set")
		}
	}

	// User: required.
	if C.User == "" {
		return errors.New("Config option user is not set")
	}

	// Password: required.
	if C.Password == "" {
		return errors.New("Config option password is not set")
	}

	// FilenamePrefix: default.
	if C.FilenamePrefix == "" {
		C.FilenamePrefix = defaultFilenamePrefix

		log.Warnf("Config option filename_prefix is not set, fallback to %s", C.FilenamePrefix)
	}

	// DefaultCompression: default and validate. If compression_algorithm is set (deprecated), use it as fallback.
	if C.DefaultCompression == "" {
		if C.CompressionAlgorithm != "" {
			C.DefaultCompression = C.CompressionAlgorithm
		} else {
			C.DefaultCompression = defaultDefaultCompression
		}
	}

	switch C.DefaultCompression {
	case "gzip", "pgzip", "bzip2", "xz", "lz4", "zstd":
		// No-op: compression algorithm validated.
	default:
		return fmt.Errorf("Config option default_compression must be gzip, bzip2, zstd, lz4, xz or pgzip, got %s", C.DefaultCompression)
	}

	// Compile exclude patterns.
	excludePatterns = make([]*regexp.Regexp, 0, len(C.ExcludePatterns))

	for _, pattern := range C.ExcludePatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Warnf("Invalid exclude pattern %q: %v", pattern, err)

			continue
		}

		excludePatterns = append(excludePatterns, re)
	}

	return nil
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
