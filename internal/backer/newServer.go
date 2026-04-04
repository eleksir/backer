package backer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"backer/internal/log"
)

// isTLSError checks if the given error is TLS-related.
// It checks for both TLS-specific error types and error messages containing "tls:".
func isTLSError(err error) bool {
	if err == nil {
		return false
	}

	var alertErr *tls.AlertError
	if errors.As(err, &alertErr) && alertErr != nil {
		return true
	}

	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) && certErr != nil {
		return true
	}

	var echErr *tls.ECHRejectionError
	if errors.As(err, &echErr) && echErr != nil {
		return true
	}

	var recordErr *tls.RecordHeaderError
	if errors.As(err, &recordErr) && recordErr != nil {
		return true
	}

	return strings.HasPrefix(err.Error(), "tls:")
}

// withServerHeader wraps a handler to add Server header to all responses.
func withServerHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "backer")
		next.ServeHTTP(w, r)
	})
}

// NewServer starts new server instance with given config file.
// It returns a serverWrapper that intercepts and classifies server errors.
func NewServer(configPath string) (*serverWrapper, error) { //nolint: revive
	_ = log.Init("info", "") // Initialize logger with defaults for early logging.

	if err := LoadConfig(configPath); err != nil {
		return nil, fmt.Errorf("Failed to load config: %w", err)
	}

	if err := log.Init(C.LogLevel, C.Log); err != nil {
		log.Errorf("Failed to initialize logger: %v", err)
	}

	log.Infof("Using %s as default compression algorithm", C.DefaultCompression)

	mux := http.NewServeMux()

	mux.HandleFunc(C.Location, func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		// No auth? Good bye.
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			log.Infof("Access denied from %s to %s (no username or password given)", r.RemoteAddr, r.RequestURI)

			return
		}

		// Both username AND password must match. Explicitly require that the whole condition
		// fails when either is incorrect, per the authentication design.
		//nolint:staticcheck // QF1001: De Morgan's law is irrelevant here.
		if !(username == C.User && password == C.Password) {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			log.Infof("Access denied from %s to %s (username and/or password incorrect)", r.RemoteAddr, r.RequestURI)

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

		var (
			archive   io.ReadCloser
			extension string
			algorithm string
		)

		requestedExt := strings.TrimPrefix(path.Ext(r.URL.Path), ".")

		algorithm = getCompressionAlgorithm(requestedExt)

		switch algorithm {
		case "bzip2":
			archive = CreateTarBzip2Stream(ctx, files)
			extension = "tar.bz2"

		case "zstd":
			archive = CreateTarZstdStream(ctx, files)
			extension = "tar.zst"

		case "lz4":
			archive = CreateTarLz4Stream(ctx, files)
			extension = "tar.lz4"

		case "xz":
			archive = CreateTarXzStream(ctx, files)
			extension = "tar.xz"

		case "pgzip":
			archive = CreateTarPgzipStream(ctx, files)
			extension = "tar.gz"

		default:
			archive = CreateTarGzStream(ctx, files)
			extension = "tar.gz"
		}

		defer func() {
			if err := archive.Close(); err != nil {
				log.Errorf("Failed to close archive reader: %v", err)
			}
		}()

		timestamp := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("%s-%s.%s", C.FilenamePrefix, timestamp, extension)

		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		w.Header().Set("Content-Type", "application/octet-stream")

		bytesWritten, err := io.Copy(w, archive)
		duration := time.Since(backupStart)

		if err != nil {
			log.Errorf("Backup failed: client=%s user=%s duration=%s error=%v", clientIP, username, duration, err)

			return
		}

		log.Infof("Backup completed: client=%s user=%s files=%d bytes=%d duration=%s", clientIP, username, len(files), bytesWritten, duration)
	})

	// HTTP mode if nohttps is enabled in config.
	// HTTPS mode if nohttps is disabled in config.
	if C.NoHTTPS {
		return &serverWrapper{
			Server: &http.Server{
				Addr:              fmt.Sprintf("%s:%d", C.Address, C.Port),
				Handler:           withServerHeader(mux),
				ReadHeaderTimeout: readHeaderTimeout,
				WriteTimeout:      time.Duration(C.BackupTimeout) * time.Minute,
				ErrorLog:          log.DebugLogger(),
			},
		}, nil
	}

	return &serverWrapper{
		Server: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", C.Address, C.Port),
			Handler:           withServerHeader(mux),
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      time.Duration(C.BackupTimeout) * time.Minute,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
			ErrorLog: log.DebugLogger(),
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
	buf := make([]byte, copyBufferSize)

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

// getCompressionAlgorithm determines which compression algorithm to use based on the requested extension.
// If the extension is recognized, it returns the corresponding algorithm; otherwise, it returns the default.
// For .tar.gz extension, pgzip is always used for parallel compression.
func getCompressionAlgorithm(requestedExt string) string {
	switch requestedExt {
	case "tar.bz2":
		return "bzip2"
	case "tar.zst":
		return "zstd"
	case "tar.lz4":
		return "lz4"
	case "tar.xz":
		return "xz"
	case "tar.gz":
		return "pgzip"
	default:
		return C.DefaultCompression
	}
}

type (
	// serverWrapper wraps [http.Server] to intercept and classify errors.
	serverWrapper struct {
		*http.Server
	}
)

// Serve starts the server and classifies any errors returned.
// TLS-related errors are logged at debug level, other errors at warn level.
func (sw *serverWrapper) Serve() error {
	err := sw.ListenAndServe()

	if err != nil {
		if isTLSError(err) {
			log.Debugf("Server error: %v", err)
		} else {
			log.Warnf("Server error: %v", err)
		}
	}

	return err
}

// ServeTLS starts the TLS server and classifies any errors returned.
func (sw *serverWrapper) ServeTLS(certFile, keyFile string) error {
	err := sw.ListenAndServeTLS(certFile, keyFile)

	if err != nil {
		if isTLSError(err) {
			log.Debugf("Server TLS error: %v", err)
		} else {
			log.Warnf("Server TLS error: %v", err)
		}
	}

	return err
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
