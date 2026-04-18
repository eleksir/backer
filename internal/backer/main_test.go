package backer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const testDataPath = "../../test_data"

func TestMain(m *testing.M) {
	if err := setupTestData(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to setup test data: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := cleanupTestData(); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Failed to cleanup test data: %v\n", err)
	}

	os.Exit(code)
}

func setupTestData() error {
	testDir := filepath.Join(testDataPath, "test1")

	dirs := []string{
		filepath.Join(testDir, "foo", "mydir"),
		filepath.Join(testDir, "bar", "goodbye.txt"),
		filepath.Join(testDir, "hardlinks"),
		filepath.Join(testDir, "symlinks"),
		filepath.Join(testDir, "foo", "empty_dir"),
		filepath.Join(testDataPath, "tmp"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("Failed to create directory %s: %w", dir, err)
		}
	}

	files := map[string]string{
		filepath.Join(testDir, "foo", "hello_breakfast.txt"):   "hello breakfast content",
		filepath.Join(testDir, "foo", "some_text.txt"):         "some text content",
		filepath.Join(testDir, "foo", "mydir", "myfile.txt"):   "my file content",
		filepath.Join(testDir, "bar", "hello.txt"):             "hello content",
		filepath.Join(testDir, "bar", "goodbye.txt", "text.txt"): "text content",
		filepath.Join(testDir, "hardlinks", "original.txt"):    "original content",
		filepath.Join(testDir, "symlinks", "target.txt"):       "target content",
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("Failed to write file %s: %w", path, err)
		}
	}

	orig := filepath.Join(testDir, "hardlinks", "original.txt")
	link1 := filepath.Join(testDir, "hardlinks", "hardlink1.txt")
	if err := os.Link(orig, link1); err != nil {
		return fmt.Errorf("Failed to create hardlink %s -> %s: %w", link1, orig, err)
	}

	link2 := filepath.Join(testDir, "hardlinks", "hardlink2.txt")
	if err := os.Link(orig, link2); err != nil {
		return fmt.Errorf("Failed to create hardlink %s -> %s: %w", link2, orig, err)
	}

	target := filepath.Join(testDir, "symlinks", "target.txt")
	link := filepath.Join(testDir, "symlinks", "link.txt")
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("Failed to create symlink %s -> %s: %w", link, target, err)
	}

	return nil
}

func cleanupTestData() error {
	testDir := filepath.Join(testDataPath, "test1")

	if err := os.RemoveAll(testDir); err != nil {
		return fmt.Errorf("Failed to remove test directory %s: %w", testDir, err)
	}

	tmpDir := filepath.Join(testDataPath, "tmp")
	if err := os.RemoveAll(tmpDir); err != nil {
		return fmt.Errorf("Failed to remove tmp directory %s: %w", tmpDir, err)
	}

	return nil
}

func GetTestDataPath() string {
	return testDataPath
}

func GetTestFilesFromDirectories(ctx context.Context, directories []string) ([]string, error) {
	return GetFilesFromDirectories(ctx, directories)
}