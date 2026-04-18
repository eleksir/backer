package backer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Reset global config
	C = Config{}

	testDataPath := filepath.Join(t.TempDir(), "test_data")
	if err := os.MkdirAll(filepath.Join(testDataPath, "test1", "foo"), 0755); err != nil {
		t.Fatalf("Failed to create test directory foo: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(testDataPath, "test1", "bar"), 0755); err != nil {
		t.Fatalf("Failed to create test directory bar: %v", err)
	}

	// Create test config file
	testConfigPath := filepath.Join(testDataPath, "test_config.json")
	testStruct := struct {
		Address     string   `json:"address"`
		Port        int      `json:"port"`
		Directories []string `json:"directories"`
		Location    string   `json:"location"`
		User        string   `json:"user"`
		Password    string   `json:"password"`
		NoHTTPS     bool     `json:"nohttps"`
	}{
		Address:     "127.0.0.1",
		Port:        8086,
		Directories: []string{filepath.Join(testDataPath, "test1", "foo"), filepath.Join(testDataPath, "test1", "bar")},
		Location:    "/archive",
		User:        "test",
		Password:    "test",
		NoHTTPS:     true,
	}
	testConfigBytes, _ := json.MarshalIndent(testStruct, "", "  ")
	if err := os.WriteFile(testConfigPath, testConfigBytes, 0644); err != nil {
		t.Fatal(err)
	}

	expectedData := Config{
		Address: "127.0.0.1",
		Port:    8086,
		Directories: []string{
			filepath.Join(testDataPath, "test1", "foo"),
			filepath.Join(testDataPath, "test1", "bar"),
		},
		Location: "/archive",
		User:     "test",
		Password: "test",
		NoHTTPS:  true,
	}

	err := LoadConfig(testConfigPath)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if C.Address != expectedData.Address {
		t.Errorf("C.Address should contain %s but contains %s", expectedData.Address, C.Address)
	}

	if C.Port != expectedData.Port {
		t.Errorf("C.Port should contain %d but containc %d", expectedData.Port, C.Port)
	}

	if C.Location != expectedData.Location {
		t.Errorf("C.StreamLocation should contain %s but contains %s", expectedData.Location, C.Location)
	}

	outputDirectoriesLen := len(C.Directories)
	expectedDataDirectoriesLen := len(expectedData.Directories)

	if outputDirectoriesLen != expectedDataDirectoriesLen {
		t.Errorf("length of C.Directories must be %d but is %d", expectedDataDirectoriesLen, outputDirectoriesLen)
	}

	for _, str := range expectedData.Directories {
		if !slices.Contains(C.Directories, str) {
			t.Errorf("C.Directories data does not contain expected string: %s", str)
		}
	}
}

// TestLoadConfigMissingUser tests that config fails when user is missing.
func TestLoadConfigMissingUser(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"port": 8086,
		"password": "test",
		"directories": ["../../test_data/test1/foo"]
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err == nil {
		t.Error("Expected error for missing user, got nil")
	}
}

// TestLoadConfigMissingPassword tests that config fails when password is missing.
func TestLoadConfigMissingPassword(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"port": 8086,
		"user": "test",
		"directories": ["../../test_data/test1/foo"]
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err == nil {
		t.Error("Expected error for missing password, got nil")
	}
}

// TestLoadConfigMissingDirectories tests that config fails when directories is missing.
func TestLoadConfigMissingDirectories(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"port": 8086,
		"user": "test",
		"password": "test"
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err == nil {
		t.Error("Expected error for missing directories, got nil")
	}
}

// TestLoadConfigInvalidPort tests that config fails with invalid port.
func TestLoadConfigInvalidPort(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"port": 99999,
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"]
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}
}

// TestLoadConfigNonExistentDirectory tests that config fails with non-existent directory.
func TestLoadConfigNonExistentDirectory(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"port": 8086,
		"user": "test",
		"password": "test",
		"directories": ["/nonexistent/path"]
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}

// TestLoadConfigHTTPSWithoutCert tests that config fails when HTTPS enabled but no cert.
func TestLoadConfigHTTPSWithoutCert(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"port": 8086,
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"],
		"nohttps": false
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err == nil {
		t.Error("Expected error for HTTPS without cert, got nil")
	}
}

// TestLoadConfigNoHTTPS tests that config succeeds with nohttps=true and no cert.
func TestLoadConfigNoHTTPS(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"port": 8086,
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"],
		"nohttps": true
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err != nil {
		t.Errorf("Expected no error with nohttps=true, got %v", err)
	}

	if !C.NoHTTPS {
		t.Error("C.NoHTTPS should be true")
	}
}

// TestLoadConfigFilenamePrefixCustom tests custom filename prefix.
func TestLoadConfigFilenamePrefixCustom(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"],
		"nohttps": true,
		"filename_prefix": "mybackup"
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if C.FilenamePrefix != "mybackup" {
		t.Errorf("C.FilenamePrefix should be mybackup, got %s", C.FilenamePrefix)
	}
}

// TestLoadConfigDefaultValues tests default values are set correctly.
func TestLoadConfigDefaultValues(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"],
		"cert": "../../test_data/example.crt",
		"key": "../../test_data/example.key"
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if C.Address != "0.0.0.0" {
		t.Errorf("C.Address should default to 0.0.0.0, got %s", C.Address)
	}

	if C.Port != 8086 {
		t.Errorf("C.Port should default to 8086, got %d", C.Port)
	}

	if C.Location != "/archive" {
		t.Errorf("C.Location should default to /archive, got %s", C.Location)
	}

	if C.LogLevel != "info" {
		t.Errorf("C.LogLevel should default to info, got %s", C.LogLevel)
	}

	if C.FilenamePrefix != "backup" {
		t.Errorf("C.FilenamePrefix should default to backup, got %s", C.FilenamePrefix)
	}

	if C.DefaultCompression != "gzip" {
		t.Errorf("C.DefaultCompression should default to gzip, got %s", C.DefaultCompression)
	}
}

// TestLoadConfigCompressionAlgorithmBzip2 tests bzip2 compression algorithm.
func TestLoadConfigCompressionAlgorithmBzip2(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"],
		"nohttps": true,
		"compression_algorithm": "bzip2"
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if C.DefaultCompression != "bzip2" {
		t.Errorf("C.DefaultCompression should be bzip2, got %s", C.DefaultCompression)
	}
}

// TestLoadConfigCompressionAlgorithmZstd tests zstd compression algorithm.
func TestLoadConfigCompressionAlgorithmZstd(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"],
		"nohttps": true,
		"compression_algorithm": "zstd"
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if C.DefaultCompression != "zstd" {
		t.Errorf("C.DefaultCompression should be zstd, got %s", C.DefaultCompression)
	}
}

// TestLoadConfigCompressionAlgorithmLz4 tests lz4 compression algorithm.
func TestLoadConfigCompressionAlgorithmLz4(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"],
		"nohttps": true,
		"compression_algorithm": "lz4"
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if C.DefaultCompression != "lz4" {
		t.Errorf("C.DefaultCompression should be lz4, got %s", C.DefaultCompression)
	}
}

// TestLoadConfigCompressionAlgorithmXz tests xz compression algorithm.
func TestLoadConfigCompressionAlgorithmXz(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"],
		"nohttps": true,
		"compression_algorithm": "xz"
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if C.DefaultCompression != "xz" {
		t.Errorf("C.DefaultCompression should be xz, got %s", C.DefaultCompression)
	}
}

// TestLoadConfigCompressionAlgorithmPgzip tests pgzip compression algorithm.
func TestLoadConfigCompressionAlgorithmPgzip(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"],
		"nohttps": true,
		"compression_algorithm": "pgzip"
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if C.DefaultCompression != "pgzip" {
		t.Errorf("C.DefaultCompression should be pgzip, got %s", C.DefaultCompression)
	}
}

// TestLoadConfigCompressionAlgorithmInvalid tests invalid compression algorithm.
func TestLoadConfigCompressionAlgorithmInvalid(t *testing.T) {
	// Reset global config
	C = Config{}
	config := `{
		"user": "test",
		"password": "test",
		"directories": ["../../test_data/test1/foo"],
		"nohttps": true,
		"compression_algorithm": "invalid"
	}`
	tmpFile := writeTempConfig(t, config)

	err := LoadConfig(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid compression algorithm, got nil")
	}
}

// writeTempConfig is a helper to write a temp config file for testing.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_config_*.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	// Reset global config before loading
	C = Config{}
	return tmpFile.Name()
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
