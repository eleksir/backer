package log

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// logFile holds the path to the current test log file for reading.
var logFilePath string

// initTestLog sets up a temp log file and returns cleanup function.
func initTestLog(t *testing.T, level string) func() {
	t.Helper()

	tmpFile, err := os.CreateTemp(t.TempDir(), "test_log_*.log")
	if err != nil {
		t.Fatal(err)
	}

	logFilePath = tmpFile.Name()
	tmpFile.Close()

	if err := Init(level, logFilePath); err != nil {
		t.Fatal(err)
	}

	return func() {
		Close()
		logFilePath = ""
	}
}

// getLogContent reads the log file and returns its contents.
func getLogContent(t *testing.T) string {
	t.Helper()

	if logFilePath == "" {
		t.Fatal("logFilePath not set")
	}

	data, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

// TestInitDefaultLevel tests Init with empty level defaults to info.
func TestInitDefaultLevel(t *testing.T) {
	cleanup := initTestLog(t, "")
	defer cleanup()

	Info("info message")
	Debug("debug message")

	content := getLogContent(t)

	if !strings.Contains(content, "info message") {
		t.Error("Expected info message in log")
	}

	if strings.Contains(content, "debug message") {
		t.Error("Debug message should not appear at info level")
	}
}

// TestInitErrorLevel tests Init with error level.
func TestInitErrorLevel(t *testing.T) {
	cleanup := initTestLog(t, "error")
	defer cleanup()

	Error("error msg")
	Warn("warn msg")
	Info("info msg")

	content := getLogContent(t)

	if !strings.Contains(content, "error msg") {
		t.Error("Expected error message in log")
	}

	if strings.Contains(content, "warn msg") {
		t.Error("Warn message should not appear at error level")
	}

	if strings.Contains(content, "info msg") {
		t.Error("Info message should not appear at error level")
	}
}

// TestInitWarnLevel tests Init with warn level.
func TestInitWarnLevel(t *testing.T) {
	cleanup := initTestLog(t, "warn")
	defer cleanup()

	Error("error msg")
	Warn("warn msg")
	Info("info msg")

	content := getLogContent(t)

	if !strings.Contains(content, "error msg") {
		t.Error("Expected error message in log")
	}

	if !strings.Contains(content, "warn msg") {
		t.Error("Expected warn message in log")
	}

	if strings.Contains(content, "info msg") {
		t.Error("Info message should not appear at warn level")
	}
}

// TestInitDebugLevel tests Init with debug level shows everything.
func TestInitDebugLevel(t *testing.T) {
	cleanup := initTestLog(t, "debug")
	defer cleanup()

	Error("error msg")
	Warn("warn msg")
	Info("info msg")
	Debug("debug msg")

	content := getLogContent(t)

	if !strings.Contains(content, "error msg") {
		t.Error("Expected error message in log")
	}

	if !strings.Contains(content, "warn msg") {
		t.Error("Expected warn message in log")
	}

	if !strings.Contains(content, "info msg") {
		t.Error("Expected info message in log")
	}

	if !strings.Contains(content, "debug msg") {
		t.Error("Expected debug message in log")
	}
}

// TestInitStderr tests Init with empty filename uses stderr.
func TestInitStderr(t *testing.T) {
	err := Init("info", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if Log != os.Stderr {
		t.Error("Expected Log to be os.Stderr")
	}
}

// TestInitInvalidPath tests Init with invalid file path returns error.
func TestInitInvalidPath(t *testing.T) {
	err := Init("info", "/nonexistent/path/to/log.log")
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

// TestErrorWithString tests Error with a string argument.
func TestErrorWithString(t *testing.T) {
	cleanup := initTestLog(t, "error")
	defer cleanup()

	Error("test error message")

	content := getLogContent(t)

	if !strings.Contains(content, "test error message") {
		t.Error("Expected error message in log")
	}
}

// TestErrorWithError tests Error with an error argument.
func TestErrorWithError(t *testing.T) {
	cleanup := initTestLog(t, "error")
	defer cleanup()

	err := errors.New("something went wrong")
	Error(err)

	content := getLogContent(t)

	if !strings.Contains(content, "something went wrong") {
		t.Error("Expected error message in log")
	}
}

// TestErrorWithMultipleArgs tests Error with multiple arguments.
func TestErrorWithMultipleArgs(t *testing.T) {
	cleanup := initTestLog(t, "error")
	defer cleanup()

	err := errors.New("disk full")
	Error("Failed to write: ", err)

	content := getLogContent(t)

	if !strings.Contains(content, "Failed to write:") {
		t.Error("Expected prefix in log")
	}

	if !strings.Contains(content, "disk full") {
		t.Error("Expected error message in log")
	}
}

// TestWarnWithString tests Warn with a string argument.
func TestWarnWithString(t *testing.T) {
	cleanup := initTestLog(t, "warn")
	defer cleanup()

	Warn("test warning")

	content := getLogContent(t)

	if !strings.Contains(content, "test warning") {
		t.Error("Expected warning message in log")
	}
}

// TestInfoWithString tests Info with a string argument.
func TestInfoWithString(t *testing.T) {
	cleanup := initTestLog(t, "info")
	defer cleanup()

	Info("test info")

	content := getLogContent(t)

	if !strings.Contains(content, "test info") {
		t.Error("Expected info message in log")
	}
}

// TestDebugWithString tests Debug with a string argument.
func TestDebugWithString(t *testing.T) {
	cleanup := initTestLog(t, "debug")
	defer cleanup()

	Debug("test debug")

	content := getLogContent(t)

	if !strings.Contains(content, "test debug") {
		t.Error("Expected debug message in log")
	}
}

// TestErrorf tests Errorf with format string.
func TestErrorf(t *testing.T) {
	cleanup := initTestLog(t, "error")
	defer cleanup()

	Errorf("Failed to read %s: %v", "config.json", errors.New("not found"))

	content := getLogContent(t)

	if !strings.Contains(content, "Failed to read config.json: not found") {
		t.Errorf("Expected formatted error in log, got: %s", content)
	}
}

// TestWarnf tests Warnf with format string.
func TestWarnf(t *testing.T) {
	cleanup := initTestLog(t, "warn")
	defer cleanup()

	Warnf("Skipping %s: %d files", "dir", 5)

	content := getLogContent(t)

	if !strings.Contains(content, "Skipping dir: 5 files") {
		t.Errorf("Expected formatted warning in log, got: %s", content)
	}
}

// TestInfof tests Infof with format string.
func TestInfof(t *testing.T) {
	cleanup := initTestLog(t, "info")
	defer cleanup()

	Infof("Server started on %s:%d", "0.0.0.0", 8086)

	content := getLogContent(t)

	if !strings.Contains(content, "Server started on 0.0.0.0:8086") {
		t.Errorf("Expected formatted info in log, got: %s", content)
	}
}

// TestDebugf tests Debugf with format string.
func TestDebugf(t *testing.T) {
	cleanup := initTestLog(t, "debug")
	defer cleanup()

	Debugf("Processing file %s (%d bytes)", "test.txt", 1024)

	content := getLogContent(t)

	if !strings.Contains(content, "Processing file test.txt (1024 bytes)") {
		t.Errorf("Expected formatted debug in log, got: %s", content)
	}
}

// TestClose tests that Close closes the log file.
func TestClose(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_log_*.log")
	if err != nil {
		t.Fatal(err)
	}

	logFilePath = tmpFile.Name()
	tmpFile.Close()

	if err := Init("info", logFilePath); err != nil {
		t.Fatal(err)
	}

	Close()

	// Verify file is closed by trying to write (should fail).
	// But we can't easily test this without reopening.
	// Just verify no panic occurred.
	logFilePath = ""
}

// TestCloseNil tests that Close doesn't panic when Log is nil.
func TestCloseNil(t *testing.T) {
	Log = nil

	// Should not panic.
	Close()
}

// TestCloseStderr tests that Close doesn't close stderr.
func TestCloseStderr(t *testing.T) {
	Log = os.Stderr

	// Should not panic and should not close stderr.
	Close()

	// Verify stderr is still usable.
	_, err := os.Stderr.WriteString("")
	if err != nil {
		t.Error("stderr should still be usable after Close()")
	}
}

// TestDebugLogger tests that DebugLogger returns a working *log.Logger.
func TestDebugLogger(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_log_*.log")
	if err != nil {
		t.Fatal(err)
	}
	logFilePath := tmpFile.Name()
	tmpFile.Close()

	if err := Init("debug", logFilePath); err != nil {
		t.Fatal(err)
	}
	defer Close()

	logger := DebugLogger()
	if logger == nil {
		t.Error("DebugLogger returned nil")
	}

	logger.Print("test debug message")

	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "test debug message") {
		t.Error("Expected debug message in log")
	}
}

// TestDebugLoggerTLS tests that TLS errors are logged at debug level.
func TestDebugLoggerTLS(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_log_*.log")
	if err != nil {
		t.Fatal(err)
	}
	logFilePath := tmpFile.Name()
	tmpFile.Close()

	if err := Init("debug", logFilePath); err != nil {
		t.Fatal(err)
	}
	defer Close()

	logger := DebugLogger()
	logger.Print("tls handshake timeout")

	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "tls handshake timeout") {
		t.Error("Expected TLS message in log")
	}
}

// TestDebugLoggerNonTLS tests that non-TLS errors are logged at warn level.
func TestDebugLoggerNonTLS(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_log_*.log")
	if err != nil {
		t.Fatal(err)
	}
	logFilePath := tmpFile.Name()
	tmpFile.Close()

	if err := Init("warn", logFilePath); err != nil {
		t.Fatal(err)
	}
	defer Close()

	logger := DebugLogger()
	logger.Print("http: connection reset by peer")

	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "http: connection reset by peer") {
		t.Error("Expected non-TLS message in log")
	}
}

// TestDebugLoggerLevelFiltering tests that log level filtering works.
func TestDebugLoggerLevelFiltering(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_log_*.log")
	if err != nil {
		t.Fatal(err)
	}
	logFilePath := tmpFile.Name()
	tmpFile.Close()

	if err := Init("info", logFilePath); err != nil {
		t.Fatal(err)
	}
	defer Close()

	logger := DebugLogger()
	logger.Print("tls handshake timeout") // TLS error at debug level - should be filtered

	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(content), "tls handshake timeout") {
		t.Error("Debug-level TLS message should be filtered at info log level")
	}
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
