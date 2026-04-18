package backer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/klauspost/compress/zstd"
)

// mkfifo creates a named pipe (FIFO).
// Uses syscall.Mkfifo which is available on Unix.
func mkfifo(path string, mode os.FileMode) error {
	return syscall.Mkfifo(path, uint32(mode))
}

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

// TestCreateTarGzStreamEmptyFile tests creation of archive with empty file.
func TestCreateTarGzStreamEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{emptyFile})
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

	if header.Size != 0 {
		t.Errorf("Expected empty file (size 0), got %d", header.Size)
	}

	if header.Typeflag != tar.TypeReg {
		t.Errorf("Expected regular file type, got %c", header.Typeflag)
	}

	data, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatalf("Failed to read content: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("Expected empty content, got %d bytes", len(data))
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

// TestCreateTarGzStreamBrokenSymlink tests handling of broken symlinks (target doesn't exist).
func TestCreateTarGzStreamBrokenSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	brokenLink := filepath.Join(tmpDir, "broken.txt")

	if err := os.Symlink("/nonexistent/target", brokenLink); err != nil {
		t.Skip("Symlinks not supported, skipping test")
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{brokenLink})
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

	// Broken symlinks are still stored - just the link target path
	if header.Linkname != "/nonexistent/target" {
		t.Errorf("Expected linkname '/nonexistent/target', got '%s'", header.Linkname)
	}
}

// TestCreateTarGzStreamSymlinkToDirectory tests symlink pointing to a directory.
func TestCreateTarGzStreamSymlinkToDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "targetdir")
	linkToDir := filepath.Join(tmpDir, "linkdir")

	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(targetDir, linkToDir); err != nil {
		t.Skip("Symlinks not supported, skipping test")
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{linkToDir})
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

	if header.Linkname != targetDir {
		t.Errorf("Expected linkname '%s', got '%s'", targetDir, header.Linkname)
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
			// Verify it's a link to the original - code strips leading '/' to avoid tar warnings
			expectedLinkname := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(originalFile)), "/")
			if header.Linkname != expectedLinkname {
				t.Errorf("Expected linkname '%s', got '%s'", expectedLinkname, header.Linkname)
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

// TestCreateTarGzStreamHardLinkFromTestData tests hard link handling with test data.
func TestCreateTarGzStreamHardLinkFromTestData(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Hard link detection not supported on Windows (inodes unavailable)")
	}

	testDataPath := filepath.Join("../../test_data", "test1", "hardlinks")
	filepaths, err := GetFilesFromDirectories(context.Background(), []string{testDataPath})
	if err != nil {
		t.Fatalf("Failed to get files from directory: %v", err)
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, filepaths)
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	type entryInfo struct {
		name      string
		typeflag byte
		linkname string
	}
	var entries []entryInfo

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar entry: %v", err)
		}

		entries = append(entries, entryInfo{
			name:      header.Name,
			typeflag:  header.Typeflag,
			linkname:  header.Linkname,
		})
		t.Logf("Entry: name=%s typeflag=%c linkname=%s", header.Name, header.Typeflag, header.Linkname)
	}

	// Check we have the expected number of entries
	if len(entries) != 4 {
		t.Errorf("Expected 4 entries (dir + 1 original + 2 hardlinks), got %d", len(entries))
	}

	// Find the original file (typeflag = TypeReg = '0') and verify hardlinks point to it
	var originalName string
	var hardlinkCount int

	for _, e := range entries {
		if e.typeflag == tar.TypeReg && e.linkname == "" {
			originalName = filepath.Base(e.name)
		}
		if e.typeflag == tar.TypeLink {
			hardlinkCount++
			// Verify link points to the original file's base name
			linkBase := filepath.Base(e.linkname)
			if linkBase != originalName && linkBase != "original.txt" {
				t.Errorf("Hard link %s points to %s, expected %s", filepath.Base(e.name), linkBase, originalName)
			}
		}
	}

	if hardlinkCount != 2 {
		t.Errorf("Expected 2 hard link entries, got %d", hardlinkCount)
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

// TestCreateTarGzStreamSpecialCharacters tests files with special characters in names.
func TestCreateTarGzStreamSpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name    string
		expect string
	}{
		{"file with spaces.txt", "file with spaces.txt"},
		{"file-with-dashes.txt", "file-with-dashes.txt"},
		{"file_with_underscores.txt", "file_with_underscores.txt"},
		{"file.multiple.dots.txt", "file.multiple.dots.txt"},
	}

	for _, tc := range testCases {
		path := filepath.Join(tmpDir, tc.name)
		if err := os.WriteFile(path, []byte(tc.name), 0644); err != nil {
			t.Fatal(err)
		}
	}

	ctx := context.Background()
	filepaths := make([]string, len(testCases))
	for i, tc := range testCases {
		filepaths[i] = filepath.Join(tmpDir, tc.name)
	}

	reader := CreateTarGzStream(ctx, filepaths)
	defer reader.Close()

	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	found := make(map[string]bool)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar entry: %v", err)
		}

		found[filepath.Base(header.Name)] = true
	}

	for _, tc := range testCases {
		if !found[tc.expect] {
			t.Errorf("Expected to find '%s' in archive", tc.expect)
		}
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

// TestCreateTarGzStreamNamedPipe tests that named pipes (FIFOs) are included in the archive.
func TestCreateTarGzStreamNamedPipe(t *testing.T) {
	tmpDir := t.TempDir()
	namedPipe := filepath.Join(tmpDir, "testpipe")

	if err := mkfifo(namedPipe, 0644); err != nil {
		t.Skip("Named pipes not supported, skipping test")
	}

	ctx := context.Background()
	reader := CreateTarGzStream(ctx, []string{namedPipe})
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

	if header.Typeflag != tar.TypeFifo {
		t.Errorf("Expected FIFO type, got %c", header.Typeflag)
	}

	// Ensure there are no more entries
	if _, err := tarReader.Next(); err != io.EOF {
		t.Error("Expected EOF after FIFO entry")
	}
}

// TestCreateTarGzStreamDeviceFile tests that device files (char and block) are included in the archive.
func TestCreateTarGzStreamDeviceFile(t *testing.T) {
	// Use /dev/null as a test device file (character device).
	devNull := "/dev/null"
	st, err := os.Lstat(devNull)
	if err != nil {
		t.Skipf("Skipping device test: %v", err)
	}

	var expectedTypeflag byte
	switch {
	case st.Mode()&os.ModeCharDevice != 0:
		expectedTypeflag = tar.TypeChar
	case st.Mode()&os.ModeDevice != 0:
		expectedTypeflag = tar.TypeBlock
	default:
		t.Skipf("Skipping: %s is not a device file", devNull)
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

	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}

	if header.Typeflag != expectedTypeflag {
		t.Errorf("Expected device type %c, got %c", expectedTypeflag, header.Typeflag)
	}

	// Ensure there are no more entries
	if _, err := tarReader.Next(); err != io.EOF {
		t.Error("Expected EOF after device entry")
	}
}

// TestCompressionLevelMappingLz4 tests that lz4 compression levels are correctly mapped.
func TestCompressionLevelMappingLz4(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{1, 0},
		{2, 0},
		{3, 0},
		{4, 1},
		{5, 1},
		{6, 1},
		{7, 2},
		{8, 2},
		{9, 2},
	}

	for _, tt := range tests {
		C = Config{CompressionLevel: tt.input}
		result := mapCompressionLevelToLz4(tt.input)
		if result != tt.expected {
			t.Errorf("mapCompressionLevelToLz4(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

// TestCompressionLevelMappingXz tests that xz compression is not configurable via level.
// xz uses default compression - this test verifies the stream works.
func TestCompressionLevelMappingXz(t *testing.T) {
	// xz doesn't support configurable compression levels in the same way.
	// It always uses default compression. This test verifies the stream is created successfully.
	C = Config{CompressionLevel: 6}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader := CreateTarXzStream(ctx, []string{testFile})
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read xz archive: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty archive")
	}
}

// TestCompressionLevelMappingBzip2 tests that bzip2 compression levels are correctly mapped.
func TestCompressionLevelMappingBzip2(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{1, 1},
		{5, 5},
		{9, 9},
	}

	for _, tt := range tests {
		C = Config{CompressionLevel: tt.input}
		result := mapCompressionLevelToBzip2(tt.input)
		if result != tt.expected {
			t.Errorf("mapCompressionLevelToBzip2(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

// TestCreateTarBzip2Stream tests bzip2 compression with configured level.
func TestCreateTarBzip2Stream(t *testing.T) {
	C = Config{CompressionLevel: 6}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content for bzip2")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader := CreateTarBzip2Stream(ctx, []string{testFile})
	defer reader.Close()

	// Verify we can read the archive
	_, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read bzip2 archive: %v", err)
	}
}

// TestCreateTarLz4Stream tests lz4 compression with configured level.
func TestCreateTarLz4Stream(t *testing.T) {
	C = Config{CompressionLevel: 6}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content for lz4")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader := CreateTarLz4Stream(ctx, []string{testFile})
	defer reader.Close()

	// Verify we can read the archive
	_, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read lz4 archive: %v", err)
	}
}

// TestCreateTarXzStream tests xz compression with configured level.
func TestCreateTarXzStream(t *testing.T) {
	C = Config{CompressionLevel: 6}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content for xz")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader := CreateTarXzStream(ctx, []string{testFile})
	defer reader.Close()

	// Verify we can read the archive
	_, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read xz archive: %v", err)
	}
}

// TestCreateTarPgzipStream tests pgzip (parallel gzip) compression.
func TestCreateTarPgzipStream(t *testing.T) {
	C = Config{CompressionLevel: 6}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content for pgzip")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader := CreateTarPgzipStream(ctx, []string{testFile})
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

	if header.Name != testFile && filepath.Base(header.Name) != "test.txt" {
		t.Errorf("Expected file name 'test.txt', got '%s'", header.Name)
	}

	data, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatalf("Failed to read file content: %v", err)
	}

	if string(data) != string(content) {
		t.Errorf("Expected content '%s', got '%s'", content, data)
	}
}

// TestCreateTarZstdStream tests zstd compression.
func TestCreateTarZstdStream(t *testing.T) {
	C = Config{CompressionLevel: 6}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content for zstd")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	reader := CreateTarZstdStream(ctx, []string{testFile})
	defer reader.Close()

	// zstd.Reader is in github.com/klauspost/compress/zstd
	zstdReader, err := zstd.NewReader(reader)
	if err != nil {
		t.Fatalf("Failed to create zstd reader: %v", err)
	}
	defer zstdReader.Close()

	tarReader := tar.NewReader(zstdReader)

	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}

	if header.Name != testFile && filepath.Base(header.Name) != "test.txt" {
		t.Errorf("Expected file name 'test.txt', got '%s'", header.Name)
	}

	data, err := io.ReadAll(tarReader)
	if err != nil {
		t.Fatalf("Failed to read file content: %v", err)
	}

	if string(data) != string(content) {
		t.Errorf("Expected content '%s', got '%s'", content, data)
	}
}

// TestZstdLevel tests zstd level mapping.
func TestZstdLevel(t *testing.T) {
	tests := []struct {
		input    int
		expected zstd.EncoderLevel
	}{
		{1, zstd.SpeedFastest},
		{2, zstd.SpeedFastest},
		{3, zstd.SpeedFastest},
		{4, zstd.SpeedDefault},
		{5, zstd.SpeedDefault},
		{6, zstd.SpeedDefault},
		{7, zstd.SpeedBetterCompression},
		{8, zstd.SpeedBetterCompression},
		{9, zstd.SpeedBetterCompression},
	}

	for _, tt := range tests {
		C = Config{CompressionLevel: tt.input}
		result := zstdLevel(tt.input)
		if result != tt.expected {
			t.Errorf("zstdLevel(%d) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
