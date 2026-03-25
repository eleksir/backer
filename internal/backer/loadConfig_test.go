package backer

import (
	"os"
	"slices"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Reset global config
	C = Config{}

	var (
		testConfigPath = "./../../test_data/test_config.json"
		expectedData   = Config{
			Address: "127.0.0.1",
			Port:    8086,
			Directories: []string{
				"../../test_data/test1/foo",
				"../../test_data/test1/bar",
			},
			Location: "/archive",
		}
	)

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

// TestLoadConfigDefaultValues tests default values are set correctly.
func TestLoadConfigDefaultValues(t *testing.T) {
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
	excludePatterns = nil
	return tmpFile.Name()
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
