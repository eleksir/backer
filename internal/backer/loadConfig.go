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
		C.Address = "0.0.0.0"

		log.Warnf("Config option address is not set, fallback to %s", C.Address)
	}

	// Port: set default, then validate range.
	if C.Port == 0 {
		C.Port = 8086

		log.Warnf("Config option port is not set, fallback to %d", C.Port)
	}

	if C.Port < 1 || C.Port > 65535 {
		return fmt.Errorf("Port must be between 1 and 65535, got %d", C.Port)
	}

	// Location: default and no range validation needed.
	if C.Location == "" {
		C.Location = "/archive"

		log.Warnf("Config option location is not set, fallback to %s", C.Location)
	}

	// Log: default only.
	if C.Log == "" {
		log.Warn("Config option log is not set, fallback to stderr")
	}

	// LogLevel: default and no range validation needed.
	if C.LogLevel == "" {
		C.LogLevel = "info"

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
		C.BackupTimeout = 60

		log.Warnf("Config option backup_timeout is not set, fallback to %d minutes", C.BackupTimeout)
	}

	if C.BackupTimeout < 1 || C.BackupTimeout > 1440 {
		return fmt.Errorf("Backup_timeout must be between 1 and 1440 minutes, got %d", C.BackupTimeout)
	}

	// CompressionLevel: set default, then validate range.
	if C.CompressionLevel == 0 {
		C.CompressionLevel = 9

		log.Warnf("Config option compression_level is not set, fallback to %d", C.CompressionLevel)
	}

	if C.CompressionLevel < 1 || C.CompressionLevel > 9 {
		return fmt.Errorf("Compression_level must be between 1 and 9, got %d", C.CompressionLevel)
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
		return errors.New("Option user is not set")
	}

	// Password: required.
	if C.Password == "" {
		return errors.New("Option password is not set")
	}

	// FilenamePrefix: default.
	if C.FilenamePrefix == "" {
		C.FilenamePrefix = "backup"

		log.Warnf("Config option filename_prefix is not set, fallback to %s", C.FilenamePrefix)
	}

	// CompressionAlgorithm: default and validate.
	if C.CompressionAlgorithm == "" {
		C.CompressionAlgorithm = "gzip"

		log.Warnf("Config option compression_algorithm is not set, fallback to %s", C.CompressionAlgorithm)
	}

	if C.CompressionAlgorithm != "gzip" && C.CompressionAlgorithm != "bzip2" && C.CompressionAlgorithm != "zstd" && C.CompressionAlgorithm != "lz4" {
		return fmt.Errorf("Compression_algorithm must be gzip, bzip2, zstd or lz4, got %s", C.CompressionAlgorithm)
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
