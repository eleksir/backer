package backer

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"backer/internal/log"
)

// withServerHeader wraps a handler to add Server header to all responses.
func withServerHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "backer")
		next.ServeHTTP(w, r)
	})
}

// NewServer starts new server instance with given config file.
func NewServer(configPath string) (*http.Server, error) {
	_ = log.Init("info", "") // Initialize logger with defaults for early logging.

	if err := LoadConfig(configPath); err != nil {
		return nil, fmt.Errorf("Failed to load config: %w", err)
	}

	if err := log.Init(C.LogLevel, C.Log); err != nil {
		log.Errorf("Failed to initialize logger: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc(C.Location, func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		// No auth? Good bye.
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			return
		}

		// Both username AND password must match. Fuck De Morgan and his bullshit laws - we explicitly require
		// that the whole condition fails. That *is* the point.
		//nolint:staticcheck // QF1001: De Morgan's law is irrelevant here.
		if !(username == C.User && password == C.Password) {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			return
		}

		clientIP := getClientIP(r)
		backupStart := time.Now()

		log.Infof("Backup started: client=%s user=%s", clientIP, username)

		ctx := r.Context()
		files, err := GetFilesFromDirectories(C.Directories)

		if err != nil {
			log.Errorf("Failed to get files: %v", err)

			w.Header().Set("Server", "backer")
			http.Error(w, "Failed to get files", http.StatusInternalServerError)

			return
		}

		tarGz := CreateTarGzStream(ctx, files)

		defer func() {
			if err := tarGz.Close(); err != nil {
				log.Errorf("Failed to close tar.gz reader: %v", err)
			}
		}()

		timestamp := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("%s-%s.tar.gz", C.FilenamePrefix, timestamp)

		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		w.Header().Set("Content-Type", "application/octet-stream")

		bytesWritten, err := io.Copy(w, tarGz)
		duration := time.Since(backupStart)

		if err != nil {
			log.Errorf("Backup failed: client=%s user=%s duration=%s error=%v", clientIP, username, duration, err)

			return
		}

		log.Infof("Backup completed: client=%s user=%s files=%d bytes=%d duration=%s", clientIP, username, len(files), bytesWritten, duration)
	})

	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", C.Address, C.Port),
		Handler:           withServerHeader(mux),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      time.Duration(C.BackupTimeout) * time.Minute,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
		},
	}, nil
}

// writeWithContext checks if context is canceled before executing fn.
// Returns ctx.Err() if context is already canceled, otherwise returns fn().
func writeWithContext(ctx context.Context, fn func() error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return fn()
}

// copyWithContext copies data from src to dst using a 32KB buffer, respecting
// context cancellation. If the context is canceled, the copy stops and returns
// ctx.Err().
func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) error {
	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}

			return readErr
		}
	}
}

// getClientIP extracts the client IP address from the request.
// It checks X-Forwarded-For header first (for proxied requests), then falls back to RemoteAddr.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
