package backer

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// normalizePath converts backslashes to forward slashes for cross-platform consistency.
func normalizePath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// normalizePathSlice normalizes all paths in a slice.
func normalizePathSlice(paths []string) []string {
	result := make([]string, len(paths))
	for i, p := range paths {
		result[i] = normalizePath(p)
	}
	return result
}

// TestGetFilesFromDirectories checks if GetFilesFromDirectories returns the proper amount of strings, correct strings and no error.
func TestGetFilesFromDirectories(t *testing.T) {
	testDataPath := "../../test_data"

	// Log the paths we're checking for debugging
	t.Logf("Checking directories: %v", []string{
		filepath.Join(testDataPath, "test1", "foo"),
		filepath.Join(testDataPath, "test1", "bar"),
	})

	var (
		input = Config{
			Directories: []string{
				filepath.Join(testDataPath, "test1", "foo"),
				filepath.Join(testDataPath, "test1", "bar"),
			},
		}
		expectedData = []string{
			filepath.Join(testDataPath, "test1", "foo"),
			filepath.Join(testDataPath, "test1", "foo", "empty_dir"),
			filepath.Join(testDataPath, "test1", "foo", "hello_breakfast.txt"),
			filepath.Join(testDataPath, "test1", "foo", "mydir"),
			filepath.Join(testDataPath, "test1", "foo", "mydir", "myfile.txt"),
			filepath.Join(testDataPath, "test1", "foo", "some_text.txt"),
			filepath.Join(testDataPath, "test1", "bar"),
			filepath.Join(testDataPath, "test1", "bar", "goodbye.txt"),
			filepath.Join(testDataPath, "test1", "bar", "goodbye.txt", "text.txt"),
			filepath.Join(testDataPath, "test1", "bar", "hello.txt"),
		}
	)

	output, err := GetFilesFromDirectories(context.Background(), input.Directories)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	outputLen := len(output)
	expectedDataLen := len(expectedData)

	if outputLen != expectedDataLen {
		t.Errorf("length of output of GetFilesFromDirectories() does not match length of expected data: %d vs %d", outputLen, expectedDataLen)
	}

	for _, filepath := range output {
		normalized := normalizePath(filepath)
		normalizedExpected := normalizePathSlice(expectedData)
		if !slices.Contains(normalizedExpected, normalized) {
			t.Errorf("expectedData data does not contain string from output: %s", filepath)
		}
	}
}

// TestGetFilesFromDirectoriesSingleDir tests with a single directory.
func TestGetFilesFromDirectoriesSingleDir(t *testing.T) {
	testDataPath := filepath.Join("../../test_data")
	output, err := GetFilesFromDirectories(context.Background(), []string{filepath.Join(testDataPath, "test1", "foo")})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}

	// Should contain the directory itself
	if !slices.Contains(output, filepath.Join(testDataPath, "test1", "foo")) {
		t.Error("Output should contain the root directory")
	}
}

// TestGetFilesFromDirectoriesEmptyDir tests with an empty directory.
func TestGetFilesFromDirectoriesEmptyDir(t *testing.T) {
	// Create temp empty dir
	tmpDir := t.TempDir()

	output, err := GetFilesFromDirectories(context.Background(), []string{tmpDir})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should contain at least the directory itself
	if len(output) != 1 {
		t.Errorf("Expected 1 entry (the directory itself), got %d", len(output))
	}
}

// TestGetFilesFromDirectoriesNonExistent tests with non-existent directory.
// Should log error but not fail - returns empty list.
func TestGetFilesFromDirectoriesNonExistent(t *testing.T) {
	testDataPath := filepath.Join("../../test_data")
	nonExistentPath := filepath.Join(testDataPath, "nonexistent", "path", "that", "does", "not", "exist")
	output, err := GetFilesFromDirectories(context.Background(), []string{nonExistentPath})

	// Should not return error, but logs error
	if err != nil {
		t.Errorf("Expected no error (logs error instead), got %v", err)
	}

	// Should return empty list
	if len(output) != 0 {
		t.Errorf("Expected empty output for non-existent directory, got %d files", len(output))
	}
}

// TestGetFilesFromDirectoriesPartialFailure tests with mix of valid and invalid directories.
func TestGetFilesFromDirectoriesPartialFailure(t *testing.T) {
	testDataPath := filepath.Join("../../test_data")
	// Create a temp directory that exists
	tmpDir := t.TempDir()

	output, err := GetFilesFromDirectories(context.Background(), []string{tmpDir, filepath.Join(testDataPath, "nonexistent", "path", "that", "does", "not", "exist")})

	// Should not return error, but logs error
	if err != nil {
		t.Errorf("Expected no error (logs error instead), got %v", err)
	}

	// Should return files from valid directory
	if len(output) == 0 {
		t.Error("Expected files from valid directory")
	}
}

// TestGetFilesFromDirectoriesMultipleDirs tests multiple directories.
func TestGetFilesFromDirectoriesMultipleDirs(t *testing.T) {
	testDataPath := filepath.Join("../../test_data")
	output, err := GetFilesFromDirectories(context.Background(), []string{
		filepath.Join(testDataPath, "test1", "foo"),
		filepath.Join(testDataPath, "test1", "bar"),
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should contain entries from both directories
	hasFoo := slices.ContainsFunc(output, func(s string) bool {
		return s == filepath.Join(testDataPath, "test1", "foo") || s == filepath.Join(testDataPath, "test1", "foo", "hello_breakfast.txt")
	})
	hasBar := slices.ContainsFunc(output, func(s string) bool {
		return s == filepath.Join(testDataPath, "test1", "bar") || s == filepath.Join(testDataPath, "test1", "bar", "hello.txt")
	})

	if !hasFoo {
		t.Error("Output should contain entries from foo directory")
	}
	if !hasBar {
		t.Error("Output should contain entries from bar directory")
	}
}

// TestGetFilesFromDirectoriesSymlink tests symlink handling.
func TestGetFilesFromDirectoriesSymlink(t *testing.T) {
	// Create temp dir with symlink
	tmpDir := t.TempDir()
	targetFile := tmpDir + "/target.txt"
	symlinkFile := tmpDir + "/link.txt"

	// Create target file
	if err := os.WriteFile(targetFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink
	if err := os.Symlink(targetFile, symlinkFile); err != nil {
		t.Skip("Symlinks not supported, skipping test")
	}

	output, err := GetFilesFromDirectories(context.Background(), []string{tmpDir})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should contain the symlink
	if !slices.ContainsFunc(output, func(s string) bool {
		return s == symlinkFile
	}) {
		t.Error("Output should contain the symlink")
	}
}

func TestIsExcluded(t *testing.T) {
	// Save original excludePatterns and restore after test
	originalPatterns := excludePatterns
	defer func() { excludePatterns = originalPatterns }()

	// Test empty patterns
	excludePatterns = nil
	if isExcluded("/some/path") {
		t.Error("Expected no exclusion with empty patterns")
	}

	// Test matching pattern
	excludePatterns = []*regexp.Regexp{
		regexp.MustCompile(`\.tmp$`),
		regexp.MustCompile(`/node_modules/`),
	}
	if !isExcluded("/tmp/foo.tmp") {
		t.Error("Expected exclusion for .tmp file")
	}
	if !isExcluded("/project/node_modules/package") {
		t.Error("Expected exclusion for node_modules path")
	}
	if isExcluded("/tmp/foo.txt") {
		t.Error("Expected no exclusion for .txt file")
	}
}

func TestGetFilesFromDirectoriesWithExcludes(t *testing.T) {
	// Save original excludePatterns and restore after test
	originalPatterns := excludePatterns
	defer func() { excludePatterns = originalPatterns }()

	// Create a temporary directory with some files
	tmpDir := t.TempDir()
	// Create files that should be included
	err := os.WriteFile(tmpDir+"/keep.txt", []byte("keep"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	// Create files that should be excluded
	err = os.WriteFile(tmpDir+"/skip.tmp", []byte("skip"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	// Create a subdirectory with file
	subDir := tmpDir + "/sub"
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(subDir+"/file.txt", []byte("file"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	// Create a .tmp file in subdirectory
	err = os.WriteFile(subDir+"/skip.tmp", []byte("skip"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Set exclude pattern for .tmp files
	excludePatterns = []*regexp.Regexp{
		regexp.MustCompile(`\.tmp$`),
	}

	output, err := GetFilesFromDirectories(context.Background(), []string{tmpDir})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Expect entries: tmpDir, keep.txt, subDir, sub/file.txt
	// Should NOT contain any .tmp files
	expected := []string{
		tmpDir,
		tmpDir + "/keep.txt",
		subDir,
		subDir + "/file.txt",
	}

	// Normalize paths for cross-platform comparison
	normalizedOutput := make([]string, len(output))
	for i, p := range output {
		normalizedOutput[i] = normalizePath(p)
	}
	normalizedExpected := make([]string, len(expected))
	for i, p := range expected {
		normalizedExpected[i] = normalizePath(p)
	}

	if len(normalizedOutput) != len(normalizedExpected) {
		t.Errorf("Expected %d entries, got %d: %v", len(normalizedExpected), len(normalizedOutput), normalizedOutput)
	}
	for _, exp := range normalizedExpected {
		found := slices.Contains(normalizedOutput, exp)
		if !found {
			t.Errorf("Expected entry %q not found", exp)
		}
	}
	// Ensure no .tmp files appear
	for _, path := range normalizedOutput {
		if strings.HasSuffix(path, ".tmp") {
			t.Errorf("Excluded file %q appeared in output", path)
		}
	}
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
