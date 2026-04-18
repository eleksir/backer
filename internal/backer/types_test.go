package backer

import (
	"testing"
)

func TestErrDirectoryScanTimeout(t *testing.T) {
	err := ErrDirectoryScanTimeout{ScannedFiles: 42}

	if err.Error() != "Directory scan timed out" {
		t.Errorf("Expected error message 'Directory scan timed out', got %q", err.Error())
	}

	if err.ScannedFiles != 42 {
		t.Errorf("Expected ScannedFiles=42, got %d", err.ScannedFiles)
	}
}