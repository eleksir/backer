package backer

import (
	"io/fs"
	"path/filepath"

	"backer/internal/log"
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
func GetFilesFromDirectories(directories []string) ([]string, error) {
	var files []string

	for _, dir := range directories {
		err := filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error { //nolint: revive
			if err != nil {
				log.Warnf("Skipping %s: %v", path, err)

				return nil
			}

			if isExcluded(path) {
				log.Debugf("Excluding: %s", path)

				return nil
			}

			files = append(files, path)

			return nil
		})

		if err != nil {
			log.Warnf("Failed to walk directory %s: %v", dir, err)
		}
	}

	return files, nil
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
