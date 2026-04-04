// Package log implements logging facility as golang slog module wrapper.
package log

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	// Log contains *[os.File] pointing to log file descriptor.
	// Protected by mu for thread-safe access, primarily to enable parallel tests.
	// In production, this is set once at startup and never modified thereafter.
	Log *os.File

	// mu protects Log during initialization and when DebugLogger is called.
	// This is necessary to allow parallel test execution where each test
	// may initialize the logger with a different log file.
	mu sync.RWMutex
)

// slogWriter implements [io.Writer] to redirect standard log to slog.
type (
	slogWriter struct {
		handler slog.Handler
	}
)

// Init sets up logger stuff.
// level can be error, warn, info, debug; if something other is supplied, info level is selected.
// Thread-safe: uses mutex to allow parallel test execution.
func Init(level, filename string) error {
	mu.Lock()
	defer mu.Unlock()

	var (
		err      error
		loglevel slog.Level
	)

	if filename != "" {
		// Close existing log file if re-initializing (e.g., in tests).
		if Log != nil && Log != os.Stderr {
			Log.Close()
		}

		Log, err = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

		if err != nil {
			return fmt.Errorf("Unable to open log file: %w", err)
		}
	} else {
		Log = os.Stderr
	}

	// no panic, no trace.
	switch level {
	case "error":
		loglevel = slog.LevelError

	case "warn":
		loglevel = slog.LevelWarn

	case "info":
		loglevel = slog.LevelInfo

	case "debug":
		loglevel = slog.LevelDebug

	default:
		loglevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		// Use the ReplaceAttr function on the handler options
		// to be able to replace any single attribute in the log output.
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr { //nolint: revive
			// Check that we are handling the time key.
			if a.Key != slog.TimeKey {
				return a
			}

			t := a.Value.Time()

			// Change the value from a time.Time to a String
			// where the string has the correct time format.
			a.Value = slog.StringValue(t.Format(time.DateTime))

			return a
		},

		Level: loglevel,
	}

	slog.SetDefault(
		slog.New(
			slog.NewTextHandler(
				Log,
				opts,
			),
		),
	)

	return err
}

// Error logs args at error level using [slog.Error()].
func Error(args ...any) {
	slog.Error(fmt.Sprint(args...))
}

// Errorf logs message on error log level and allows to supply arguments in printf() style.
func Errorf(format string, a ...any) {
	slog.Error(fmt.Sprintf(format, a...))
}

// Warn logs args at warn level using [slog.Warn()].
func Warn(args ...any) {
	slog.Warn(fmt.Sprint(args...))
}

// Warnf logs message on warn log level and allows to supply arguments in printf() style.
func Warnf(format string, a ...any) {
	slog.Warn(fmt.Sprintf(format, a...))
}

// Info logs args at info level using [slog.Info()].
func Info(args ...any) {
	slog.Info(fmt.Sprint(args...))
}

// Infof logs message on info log level and allows to supply arguments in printf() style.
func Infof(format string, a ...any) {
	slog.Info(fmt.Sprintf(format, a...))
}

// Debug logs args at debug level using [slog.Debug()].
func Debug(args ...any) {
	slog.Debug(fmt.Sprint(args...))
}

// Debugf logs message on debug log level and allows to supply arguments in printf() style.
func Debugf(format string, a ...any) {
	slog.Debug(fmt.Sprintf(format, a...))
}

// DebugLogger returns a standard library logger that writes to debug level.
// This is useful for assigning to http.Server.ErrorLog to move TLS handshake
// and other client-side errors to debug level.
func DebugLogger() *log.Logger {
	mu.RLock()
	defer mu.RUnlock()

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewTextHandler(Log, opts)

	return log.New(&slogWriter{handler: handler}, "", 0)
}

// Write implements [io.Writer]. It differentiates TLS/SSL errors from other errors.
// TLS-related errors are logged at debug level, while other errors (handler issues,
// unexpected behavior) are logged at warn level.
func (w *slogWriter) Write(p []byte) (int, error) {
	msg := string(p)

	isTLSError := strings.Contains(msg, "tls") ||
		strings.Contains(msg, "ssl") ||
		strings.Contains(msg, "TLS") ||
		strings.Contains(msg, "SSL") ||
		strings.Contains(msg, "handshake") ||
		strings.Contains(msg, "certificate")

	level := slog.LevelWarn
	if isTLSError {
		level = slog.LevelDebug
	}

	record := slog.NewRecord(time.Now(), level, msg, 0)

	_ = w.handler.Handle(context.TODO(), record)

	return len(p), nil
}

// Fatal logs args at error level and quits with status code 1.
func Fatal(args ...any) {
	slog.Error(fmt.Sprint(args...))
	Close()
	os.Exit(1)
}

// Fatalf logs message on error log level and allows to supply arguments in printf() style. After that it quits with
// status code 1.
func Fatalf(format string, a ...any) {
	slog.Error(fmt.Sprintf(format, a...))
	Close()
	os.Exit(1)
}

// Close closes log file descriptor.
// Thread-safe: uses mutex to coordinate with Init and DebugLogger.
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if Log == nil || Log == os.Stderr {
		return
	}

	if err := Log.Close(); err != nil {
		msg := fmt.Sprintln("Failed to close log file:", err)
		slog.Error(msg)
	}

	Log = nil
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
