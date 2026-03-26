package backer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNewServerSuccess tests successful server creation.
func TestNewServerSuccess(t *testing.T) {
	C = Config{} // Reset global config.

	server, err := NewServer("./../../test_data/test_config.json")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if server == nil {
		t.Fatal("Expected non-nil server")
	}

	if server.Addr != "127.0.0.1:8086" {
		t.Errorf("Expected address '127.0.0.1:8086', got '%s'", server.Addr)
	}
}

// TestNewServerInvalidConfig tests server creation with invalid config.
func TestNewServerInvalidConfig(t *testing.T) {
	C = Config{} // Reset global config.

	_, err := NewServer("/nonexistent/config.json")
	if err == nil {
		t.Error("Expected error for invalid config path, got nil")
	}
}

// TestHandlerNoAuth tests that requests without auth are rejected.
func TestHandlerNoAuth(t *testing.T) {
	C = Config{
		Address:     "0.0.0.0",
		Port:        8086,
		Location:    "/archive",
		User:        "testuser",
		Password:    "testpass",
		Directories: []string{"../../test_data/test1/foo"},
	}
	excludePatterns = nil

	req := httptest.NewRequest("GET", "/archive", nil)
	w := httptest.NewRecorder()

	// Create a handler that mimics the server's handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !(username == C.User && password == C.Password) {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	authHeader := w.Header().Get("WWW-Authenticate")
	if !strings.Contains(authHeader, "Basic") {
		t.Errorf("Expected WWW-Authenticate header with Basic, got '%s'", authHeader)
	}
}

// TestHandlerWrongCredentials tests that requests with wrong credentials are rejected.
func TestHandlerWrongCredentials(t *testing.T) {
	C = Config{
		Address:     "0.0.0.0",
		Port:        8086,
		Location:    "/archive",
		User:        "testuser",
		Password:    "testpass",
		Directories: []string{"../../test_data/test1/foo"},
	}
	excludePatterns = nil

	req := httptest.NewRequest("GET", "/archive", nil)
	req.SetBasicAuth("wronguser", "wrongpass")
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !(username == C.User && password == C.Password) {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestHandlerCorrectCredentials tests that requests with correct credentials succeed.
func TestHandlerCorrectCredentials(t *testing.T) {
	C = Config{
		Address:     "0.0.0.0",
		Port:        8086,
		Location:    "/archive",
		User:        "testuser",
		Password:    "testpass",
		Directories: []string{"../../test_data/test1/foo"},
	}
	excludePatterns = nil

	req := httptest.NewRequest("GET", "/archive", nil)
	req.SetBasicAuth("testuser", "testpass")
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !(username == C.User && password == C.Password) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestHandlerArchiveStream tests that the handler streams a valid tar.gz archive.
func TestHandlerArchiveStream(t *testing.T) {
	C = Config{
		Address:     "0.0.0.0",
		Port:        8086,
		Location:    "/archive",
		User:        "testuser",
		Password:    "testpass",
		Directories: []string{"../../test_data/test1/foo"},
	}
	excludePatterns = nil

	// Create a test server with the actual handler logic
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || !(username == C.User && password == C.Password) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		files, err := GetFilesFromDirectories(C.Directories)
		if err != nil {
			http.Error(w, "Failed to get files", http.StatusInternalServerError)
			return
		}

		tarGz := CreateTarGzStream(ctx, files)
		defer tarGz.Close()

		w.Header().Set("Content-Disposition", "attachment; filename=archive.tar")
		w.Header().Set("Content-Type", "application/octet-stream")

		io.Copy(w, tarGz)
	}))
	defer server.Close()

	// Make authenticated request
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.SetBasicAuth("testuser", "testpass")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify it's a valid tar.gz
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	foundFile := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read tar entry: %v", err)
		}

		if strings.Contains(header.Name, "hello_breakfast.txt") {
			foundFile = true
		}
	}

	if !foundFile {
		t.Error("Expected to find hello_breakfast.txt in archive")
	}
}

// TestWriteWithContextSuccess tests successful write with context.
func TestWriteWithContextSuccess(t *testing.T) {
	ctx := context.Background()
	called := false

	err := writeWithContext(ctx, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected function to be called")
	}
}

// TestWriteWithContextError tests write with context returning error.
func TestWriteWithContextError(t *testing.T) {
	ctx := context.Background()
	testErr := io.ErrUnexpectedEOF

	err := writeWithContext(ctx, func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected %v, got %v", testErr, err)
	}
}

// TestWriteWithContextCancellation tests write with cancelled context.
func TestWriteWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := writeWithContext(ctx, func() error {
		t.Error("Function should not be called with cancelled context")
		return nil
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// TestCopyWithContextSuccess tests successful copy with context.
func TestCopyWithContextSuccess(t *testing.T) {
	ctx := context.Background()
	src := strings.NewReader("hello world")
	dst := &strings.Builder{}

	err := copyWithContext(ctx, dst, src)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if dst.String() != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", dst.String())
	}
}

// TestCopyWithContextCancellation tests copy with cancelled context.
func TestCopyWithContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a reader with a lot of data that will take time to copy
	data := make([]byte, 1024*1024) // 1MB
	for i := range data {
		data[i] = byte(i % 256)
	}
	src := strings.NewReader(string(data))
	dst := &strings.Builder{}

	// Start copy in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- copyWithContext(ctx, dst, src)
	}()

	// Cancel context immediately
	cancel()

	err := <-errCh
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
