package backer

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	backerlog "backer/internal/log"
)

// isExcluded checks if path matches any of the configured exclude patterns.
// Paths are normalized to forward slashes for cross-platform regex matching.
func isExcluded(path string) bool {
	normalized := filepath.ToSlash(path)

	for _, re := range excludePatterns {
		if re.MatchString(normalized) {
			return true
		}
	}

	return false
}

// GetFilesFromDirectories makes a file list of given directories.
// Logs errors for directories that don't exist or can't be accessed but continues with valid directories.
// Takes context with timeout from config dir_scan_timeout setting.
func GetFilesFromDirectories(ctx context.Context, directories []string) ([]string, error) {
	var files []string

	for _, dir := range directories {
		// Check directory exists before walking.
		if _, err := os.Stat(dir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				backerlog.Errorf("Configured backup directory not found: %s", dir)
			} else {
				backerlog.Errorf("Failed to access configured directory %s: %v", dir, err)
			}

			continue
		}

		err := filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error { //nolint: revive
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err != nil {
				backerlog.Warnf("Skipping %s: %v", path, err)

				return nil
			}

			if isExcluded(path) {
				backerlog.Debugf("Excluding: %s", path)

				return nil
			}

			files = append(files, path)

			return nil
		})

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				backerlog.Errorf("Directory scan timed out after scanning %d files", len(files))

				return files, ErrDirectoryScanTimeout{ScannedFiles: len(files)}
			}

			backerlog.Warnf("Failed to walk directory %s: %v", dir, err)
		}
	}

	return files, nil
}

// GetFilesFromDirectoriesWithTimeout makes a file list of given directories with timeout.
// Convenience wrapper that creates context from timeout in minutes.
func GetFilesFromDirectoriesWithTimeout(timeoutMinutes int, directories []string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	return GetFilesFromDirectories(ctx, directories)
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
