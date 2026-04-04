package backer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestCreateTarGzStreamBasic tests basic tar.gz creation with a single file.
func TestCreateTarGzStreamBasic(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{testFile})
	defer reader.Close()

	// Decompress and read
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Read the file entry
	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}

	if header.Name != testFile && filepath.Base(header.Name) != "test.txt" {
		t.Errorf("Expected file name 'test.txt', got '%s'", header.Name)
	}

	if header.Typeflag != tar.TypeReg {
		t.Errorf("Expected regular file type, got %d", header.Typeflag)
	}

	// Read content
	data, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatalf("Failed to read file content: %v", err)
	}

	if string(data) != string(content) {
		t.Errorf("Expected content '%s', got '%s'", content, data)
	}

	// Should be no more entries
	if _, err := tarReader.Next(); err != io.EOF {
		t.Error("Expected EOF after single file")
	}
}

// TestCreateTarGzStreamMultipleFiles tests with multiple files.
func TestCreateTarGzStreamMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple files
	files := []string{
		filepath.Join(tmpDir, "file1.txt"),
		filepath.Join(tmpDir, "file2.txt"),
		filepath.Join(tmpDir, "file3.txt"),
	}

	for i, f := range files {
		if err := os.WriteFile(f, []byte{byte('0' + i)}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, files)
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	count := 0
	for {
		_, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar entry: %v", err)
		}
		count++
	}

	if count != 3 {
		t.Errorf("Expected 3 entries, got %d", count)
	}
}

// TestCreateTarGzStreamDirectory tests directory handling.
func TestCreateTarGzStreamDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{subDir})
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}

	if header.Typeflag != tar.TypeDir {
		t.Errorf("Expected directory type, got %d", header.Typeflag)
	}
}

// TestCreateTarGzStreamSymlink tests symlink handling.
func TestCreateTarGzStreamSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "target.txt")
	symlinkFile := filepath.Join(tmpDir, "link.txt")

	if err := os.WriteFile(targetFile, []byte("target content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(targetFile, symlinkFile); err != nil {
		t.Skip("Symlinks not supported, skipping test")
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{symlinkFile})
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}

	if header.Typeflag != tar.TypeSymlink {
		t.Errorf("Expected symlink type, got %c", header.Typeflag)
	}

	if header.Linkname != targetFile {
		t.Errorf("Expected linkname '%s', got '%s'", targetFile, header.Linkname)
	}
}

// TestCreateTarGzStreamHardLink tests hard link handling - should deduplicate content.
func TestCreateTarGzStreamHardLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Hard link detection not supported on Windows (inodes unavailable)")
	}

	tmpDir := t.TempDir()

	// Create original file
	originalFile := filepath.Join(tmpDir, "original.txt")
	if err := os.WriteFile(originalFile, []byte("shared content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create hard link
	hardLinkFile := filepath.Join(tmpDir, "hardlink.txt")
	if err := os.Link(originalFile, hardLinkFile); err != nil {
		t.Skip("Hard links not supported, skipping test")
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{originalFile, hardLinkFile})
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	entries := 0
	linkCount := 0

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar entry: %v", err)
		}

		entries++
		t.Logf("Entry %d: name=%s typeflag=%c linkname=%s", entries, header.Name, header.Typeflag, header.Linkname)

		if header.Typeflag == tar.TypeLink {
			linkCount++
			// Verify it's a link to the original
			if header.Linkname != originalFile && header.Linkname != filepath.ToSlash(originalFile) {
				t.Errorf("Expected linkname '%s', got '%s'", originalFile, header.Linkname)
			}
		}
	}

	// Should have 2 entries: one file content, one as hard link
	if entries != 2 {
		t.Errorf("Expected 2 entries, got %d", entries)
	}

	// One should be a hard link (TypeLink)
	if linkCount != 1 {
		t.Errorf("Expected 1 hard link entry, got %d", linkCount)
	}
}

// TestCreateTarGzStreamCancellation tests context cancellation.
func TestCreateTarGzStreamCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel before reading
	cancel()

	reader := CreateTarGzStream(ctx, []string{testFile})
	defer reader.Close()

	// Read all - should complete without panic
	data, err := io.ReadAll(reader)
	if err != nil {
		// Error is expected due to cancellation
		t.Logf("Got expected error: %v", err)
	}

	// We should get some data or an error, but not panic
	_ = data
}

// TestCreateTarGzStreamNonExistentFile tests handling of non-existent files.
func TestCreateTarGzStreamNonExistentFile(t *testing.T) {
	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{"/nonexistent/path/file.txt"})
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Should get EOF immediately since the file doesn't exist and is skipped
	_, err = tarReader.Next()
	if err != io.EOF {
		t.Errorf("Expected EOF for non-existent file, got: %v", err)
	}
}

// TestCreateTarGzStreamEmptyList tests with empty file list.
func TestCreateTarGzStreamEmptyList(t *testing.T) {
	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{})
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Should get EOF immediately for empty list
	_, err = tarReader.Next()
	if err != io.EOF {
		t.Errorf("Expected EOF for empty list, got: %v", err)
	}
}

// TestCreateTarGzStreamNestedFiles tests with nested directory structure.
func TestCreateTarGzStreamNestedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	nestedFile := filepath.Join(nestedDir, "deep.txt")
	if err := os.WriteFile(nestedFile, []byte("deep content"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	// Pass the nested file directly, not the root
	reader := CreateTarGzStream(ctx, []string{nestedFile})
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	foundDeepFile := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar entry: %v", err)
		}

		if filepath.Base(header.Name) == "deep.txt" {
			foundDeepFile = true

			// Verify content
			data, err := io.ReadAll(tarReader)
			if err != nil {
				t.Fatalf("Failed to read content: %v", err)
			}
			if string(data) != "deep content" {
				t.Errorf("Expected 'deep content', got '%s'", data)
			}
		}
	}

	if !foundDeepFile {
		t.Error("Expected to find deep.txt in archive")
	}
}

// TestCreateTarGzStreamDeviceFile tests that device files are included in the archive.
func TestCreateTarGzStreamDeviceFile(t *testing.T) {
	// Use /dev/null as a test device file (character device).
	devNull := "/dev/null"
	st, err := os.Lstat(devNull)
	if err != nil {
		t.Skipf("Skipping device test: %v", err)
	}
	if st.Mode()&os.ModeCharDevice == 0 {
		t.Skipf("Skipping: %s is not a character device", devNull)
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{devNull})
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Read the device entry
	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}

	if header.Typeflag != tar.TypeChar {
		t.Errorf("Expected character device type, got %c", header.Typeflag)
	}

	// Check that the name matches the file path (without leading slash)
	expectedName := devNull
	if expectedName[0] == '/' {
		expectedName = expectedName[1:]
	}
	if header.Name != expectedName {
		t.Errorf("Expected header name %q, got %q", expectedName, header.Name)
	}

	// Ensure there are no more entries
	if _, err := tarReader.Next(); err != io.EOF {
		t.Error("Expected EOF after device entry")
	}
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
