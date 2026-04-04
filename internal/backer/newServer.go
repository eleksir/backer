package backer

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
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
// It returns a ServerWrapper that intercepts and classifies server errors.
func NewServer(configPath string) (*ServerWrapper, error) {
	_ = log.Init("info", "") // Initialize logger with defaults for early logging.

	if err := LoadConfig(configPath); err != nil {
		return nil, fmt.Errorf("Failed to load config: %w", err)
	}

	if err := log.Init(C.LogLevel, C.Log); err != nil {
		log.Errorf("Failed to initialize logger: %v", err)
	}

	log.Infof("Using %s as default compression algorithm", C.DefaultCompression)

	mux := http.NewServeMux()

	backupHandler := func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		// No auth? Good bye.
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			log.Infof("Access denied from %s to %s (no username or password given)", r.RemoteAddr, r.RequestURI)

			return
		}

		// Both username AND password must match. Use timing-safe comparison to prevent timing attacks.
		if !timingSafeMatch(username, C.User) || !timingSafeMatch(password, C.Password) {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			log.Infof("Access denied from %s to %s (username and/or password incorrect)", r.RemoteAddr, r.RequestURI)

			return
		}

		clientIP := getClientIP(r)
		backupStart := time.Now()

		log.Infof("Backup started: client=%s user=%s", clientIP, username)

		// Create context with dir_scan_timeout from request context.
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(C.DirScanTimeout)*time.Minute)
		defer cancel()

		files, err := GetFilesFromDirectories(ctx, C.Directories)

		if err != nil {
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				w.Header().Set("Server", "backer")
				http.Error(w, "Directory scan timed out", http.StatusRequestTimeout)

				log.Errorf("Directory scan timed out: %v", err)

			default:
				log.Errorf("Failed to get files: %v", err)

				w.Header().Set("Server", "backer")
				http.Error(w, "Failed to get files", http.StatusInternalServerError)
			}

			return
		}

		if len(files) == 0 {
			log.Warnf("No files found to backup in configured directories")
		}

		var (
			archive   io.ReadCloser
			extension string
			algorithm string
		)

		requestedExt := r.URL.Path

		algorithm = getCompressionAlgorithm(requestedExt)

		switch algorithm {
		case "bzip2":
			archive = CreateTarBzip2Stream(ctx, files)
			extension = "tar.bz2"

		case "zstd":
			archive = CreateTarZstdStream(ctx, files)
			if strings.HasSuffix(requestedExt, ".zst") {
				extension = "tar.zst"
			} else {
				extension = "tar.zstd"
			}

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

		timestamp := backupStart.Format("20060102-150405")
		filename := fmt.Sprintf("%s-%s.%s", C.FilenamePrefix, timestamp, extension)

		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		w.Header().Set("Content-Type", "application/octet-stream")

		bytesWritten, md5Hash, err := copyWithContext(ctx, w, archive)
		if err != nil {
			duration := time.Since(backupStart)

			if errors.Is(err, context.Canceled) {
				log.Infof("Backup canceled: client=%s user=%s duration=%s", clientIP, username, duration)

				return
			}

			if isPipeClosedError(err) {
				log.Infof("Client disconnected during backup: client=%s user=%s", clientIP, username)

				return
			}

			log.Errorf("Backup failed: client=%s user=%s duration=%s error=%v", clientIP, username, duration, err)

			return
		}

		log.Infof("Backup completed: client=%s user=%s files=%d bytes=%d md5=%s duration=%s", clientIP, username, len(files), bytesWritten, md5Hash, time.Since(backupStart))
	}

	mux.Handle(C.Location, http.HandlerFunc(backupHandler))

	extensions := []string{".tar.gz", ".tar.xz", ".tar.bz2", ".tar.lz4", ".tar.zst", ".tar.zstd"}
	for _, ext := range extensions {
		muxPath := C.Location + ext
		mux.Handle(muxPath, http.HandlerFunc(backupHandler))
	}

	// HTTP mode if nohttps is enabled in config.
	// HTTPS mode if nohttps is disabled in config.
	if C.NoHTTPS {
		srv := &ServerWrapper{
			Server: &http.Server{
				Addr:              fmt.Sprintf("%s:%d", C.Address, C.Port),
				Handler:           withServerHeader(mux),
				ReadHeaderTimeout: readHeaderTimeout,
				WriteTimeout:      time.Duration(C.BackupTimeout) * time.Minute,
				IdleTimeout:       idleTimeout,
				MaxHeaderBytes:    maxHeaderBytes,
				ErrorLog:          log.DebugLogger(),
			},
		}
		srv.SetKeepAlivesEnabled(false)

		return srv, nil
	}

	srv := &ServerWrapper{
		Server: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", C.Address, C.Port),
			Handler:           withServerHeader(mux),
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      time.Duration(C.BackupTimeout) * time.Minute,
			IdleTimeout:       idleTimeout,
			MaxHeaderBytes:    maxHeaderBytes,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
			ErrorLog: log.DebugLogger(),
		},
	}
	srv.SetKeepAlivesEnabled(false)

	return srv, nil
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
// context cancellation. Returns bytes written, MD5 hash (hex string), and any error encountered.
//
//nolint:gosec
func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, string, error) {
	buf := make([]byte, copyBufferSize)

	var bytesWritten int64

	hasher := md5.New()

	for {
		select {
		case <-ctx.Done():
			return bytesWritten, "", ctx.Err()
		default:
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			select {
			case <-ctx.Done():
				return bytesWritten, "", ctx.Err()
			default:
			}

			if _, werr := hasher.Write(buf[:n]); werr != nil {
				return bytesWritten, "", werr
			}

			if _, werr := dst.Write(buf[:n]); werr != nil {
				return bytesWritten, "", werr
			}

			bytesWritten += int64(n)
		}

		if readErr != nil {
			if readErr == io.EOF {
				hash := hex.EncodeToString(hasher.Sum(nil))

				return bytesWritten, hash, nil
			}

			return bytesWritten, "", readErr
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
// The input can be either a full path like "/archive.tar.xz" or just the extension like "tar.xz" or "xz".
func getCompressionAlgorithm(input string) string {
	// Extract extension from full path (e.g., "/archive.tar.xz" -> ".xz").
	// or use the input directly if it's just an extension (e.g., "tar.xz" -> ".xz").
	ext := path.Ext(input)
	if ext == "" {
		ext = "." + input
	}

	switch ext {
	case ".tar.bz2":
		return "bzip2"
	case ".tar.zstd":
		return "zstd"
	case ".tar.zst":
		return "zstd"
	case ".tar.lz4":
		return "lz4"
	case ".tar.xz":
		return "xz"
	case ".tar.gz":
		return "pgzip"
	case ".xz":
		return "xz"
	case ".bz2":
		return "bzip2"
	case ".zst":
		return "zstd"
	case ".zstd":
		return "zstd"
	case ".lz4":
		return "lz4"
	case ".gz":
		return "pgzip"
	default:
		return C.DefaultCompression
	}
}

type (
	// ServerWrapper wraps [http.Server] to intercept and classify errors.
	ServerWrapper struct {
		*http.Server
	}
)

// Serve starts the server and classifies any errors returned.
// TLS-related errors are logged at debug level, other errors at warn level.
func (sw *ServerWrapper) Serve() error {
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
func (sw *ServerWrapper) ServeTLS(certFile, keyFile string) error {
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

// timingSafeMatch performs constant-time comparison to prevent timing attacks.
// Returns 1 if the strings are equal, 0 otherwise.
func timingSafeMatch(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
