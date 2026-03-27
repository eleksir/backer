package backer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"backer/internal/log"

	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4"
	"github.com/ulikunitz/xz"
)

// CreateTarGzStream takes a list of files and returns a reader that reads from a tar archive containing these files.
func CreateTarGzStream(ctx context.Context, filepaths []string) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		var pipeErr error // Track if we close pipe with error.

		// Helper to close pipe with error, only once.
		closePipeWithError := func(err error) {
			if pipeErr != nil {
				return // Already closed with error.
			}

			pipeErr = err
			pw.CloseWithError(err)
		}

		defer func() {
			// Only do normal close if we haven't closed with error.
			if pipeErr == nil {
				pw.Close()
			}
		}()

		gw, err := gzip.NewWriterLevel(pw, C.CompressionLevel)
		if err != nil {
			closePipeWithError(fmt.Errorf("gzip writer error: %w", err))

			return
		}

		defer func() {
			// Only close gzip writer if pipe is still open.
			if pipeErr == nil {
				if err := gw.Close(); err != nil {
					// Ignore pipe closed errors - reader may have closed early.
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close gzip writer: %v", err)
					}
				}
			}
		}()

		tw := tar.NewWriter(gw)

		defer func() {
			// Only close tar writer if pipe is still open.
			if pipeErr == nil {
				if err := tw.Close(); err != nil {
					// Ignore pipe closed errors - reader may have closed early.
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close tar writer: %v", err)
					}
				}
			}
		}()

		for _, fpath := range filepaths {
			var (
				err        error
				linkTarget string
			)

			// Check for cancellation early.
			select {
			case <-ctx.Done():
				closePipeWithError(ctx.Err())

				return

			default:
			}

			log.Debugf("Adding file: %s", fpath)

			st, err := os.Lstat(fpath)

			if err != nil {
				log.Warnf("Skipping %s: %v", fpath, err)

				continue
			}

			if st.Mode()&os.ModeSymlink != 0 {
				linkTarget, err = os.Readlink(fpath)

				if err != nil {
					log.Warnf("Skipping %s: readlink failed: %v", fpath, err)

					continue
				}
			}

			// Skip sockets explicitly (they cannot be backed up).
			if st.Mode()&os.ModeSocket != 0 {
				log.Debugf("Skipping socket: %s", fpath)

				continue
			}

			header, err := tar.FileInfoHeader(st, linkTarget)

			if err != nil {
				log.Warnf("Skipping %s: tar header failed: %v", fpath, err)

				continue
			}

			header.Format = tar.FormatGNU
			header.Name = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(fpath)), "/")

			mode := st.Mode()

			// Write header with cancellation awareness.
			if err := writeWithContext(ctx, func() error {
				return tw.WriteHeader(header)
			}); err != nil {
				closePipeWithError(err)

				return
			}

			// Only stream file contents for regular files.
			if mode.IsRegular() {
				f, err := os.Open(fpath)
				if err != nil {
					log.Warnf("Skipping %s: open failed: %v", fpath, err)

					continue
				}

				err = copyWithContext(ctx, tw, f)

				if e := f.Close(); e != nil {
					log.Warnf("Failed to close file %s: %v", fpath, e)
				}

				if err != nil {
					log.Warnf("Skipping %s: copy failed: %v", fpath, err)

					continue
				}
			}
		}
	}()

	return pr
}

// CreateTarBzip2Stream takes a list of files and returns a reader that reads from a bzip2-compressed tar archive.
func CreateTarBzip2Stream(ctx context.Context, filepaths []string) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		var pipeErr error

		closePipeWithError := func(err error) {
			if pipeErr != nil {
				return
			}

			pipeErr = err
			pw.CloseWithError(err)
		}

		defer func() {
			if pipeErr == nil {
				pw.Close()
			}
		}()

		bw, err := bzip2.NewWriter(pw, nil)
		if err != nil {
			closePipeWithError(fmt.Errorf("bzip2 writer error: %w", err))

			return
		}

		defer func() {
			if pipeErr == nil {
				if err := bw.Close(); err != nil {
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close bzip2 writer: %v", err)
					}
				}
			}
		}()

		tw := tar.NewWriter(bw)

		defer func() {
			if pipeErr == nil {
				if err := tw.Close(); err != nil {
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close tar writer: %v", err)
					}
				}
			}
		}()

		for _, fpath := range filepaths {
			var (
				err        error
				linkTarget string
			)

			select {
			case <-ctx.Done():
				closePipeWithError(ctx.Err())

				return

			default:
			}

			log.Debugf("Adding file: %s", fpath)

			st, err := os.Lstat(fpath)

			if err != nil {
				log.Warnf("Skipping %s: %v", fpath, err)

				continue
			}

			if st.Mode()&os.ModeSymlink != 0 {
				linkTarget, err = os.Readlink(fpath)

				if err != nil {
					log.Warnf("Skipping %s: readlink failed: %v", fpath, err)

					continue
				}
			}

			if st.Mode()&os.ModeSocket != 0 {
				log.Debugf("Skipping socket: %s", fpath)

				continue
			}

			header, err := tar.FileInfoHeader(st, linkTarget)

			if err != nil {
				log.Warnf("Skipping %s: tar header failed: %v", fpath, err)

				continue
			}

			header.Format = tar.FormatGNU
			header.Name = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(fpath)), "/")

			mode := st.Mode()

			if err := writeWithContext(ctx, func() error {
				return tw.WriteHeader(header)
			}); err != nil {
				closePipeWithError(err)

				return
			}

			if mode.IsRegular() {
				f, err := os.Open(fpath)
				if err != nil {
					log.Warnf("Skipping %s: open failed: %v", fpath, err)

					continue
				}

				err = copyWithContext(ctx, tw, f)

				if e := f.Close(); e != nil {
					log.Warnf("Failed to close file %s: %v", fpath, e)
				}

				if err != nil {
					log.Warnf("Skipping %s: copy failed: %v", fpath, err)

					continue
				}
			}
		}
	}()

	return pr
}

// zstdLevel returns the zstd encoder level based on the 1-9 compression level.
func zstdLevel(level int) zstd.EncoderLevel {
	switch {
	case level <= 3:
		return zstd.SpeedFastest
	case level <= 6:
		return zstd.SpeedDefault
	default:
		return zstd.SpeedBetterCompression
	}
}

// CreateTarZstdStream takes a list of files and returns a reader that reads from a zstd-compressed tar archive.
func CreateTarZstdStream(ctx context.Context, filepaths []string) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		var pipeErr error

		closePipeWithError := func(err error) {
			if pipeErr != nil {
				return
			}

			pipeErr = err
			pw.CloseWithError(err)
		}

		defer func() {
			if pipeErr == nil {
				pw.Close()
			}
		}()

		zw, err := zstd.NewWriter(pw, zstd.WithEncoderLevel(zstdLevel(C.CompressionLevel)))
		if err != nil {
			closePipeWithError(fmt.Errorf("zstd writer error: %w", err))

			return
		}

		defer func() {
			if pipeErr == nil {
				if err := zw.Close(); err != nil {
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close zstd writer: %v", err)
					}
				}
			}
		}()

		tw := tar.NewWriter(zw)

		defer func() {
			if pipeErr == nil {
				if err := tw.Close(); err != nil {
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close tar writer: %v", err)
					}
				}
			}
		}()

		for _, fpath := range filepaths {
			var (
				err        error
				linkTarget string
			)

			select {
			case <-ctx.Done():
				closePipeWithError(ctx.Err())

				return

			default:
			}

			log.Debugf("Adding file: %s", fpath)

			st, err := os.Lstat(fpath)

			if err != nil {
				log.Warnf("Skipping %s: %v", fpath, err)

				continue
			}

			if st.Mode()&os.ModeSymlink != 0 {
				linkTarget, err = os.Readlink(fpath)

				if err != nil {
					log.Warnf("Skipping %s: readlink failed: %v", fpath, err)

					continue
				}
			}

			if st.Mode()&os.ModeSocket != 0 {
				log.Debugf("Skipping socket: %s", fpath)

				continue
			}

			header, err := tar.FileInfoHeader(st, linkTarget)

			if err != nil {
				log.Warnf("Skipping %s: tar header failed: %v", fpath, err)

				continue
			}

			header.Format = tar.FormatGNU
			header.Name = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(fpath)), "/")

			mode := st.Mode()

			if err := writeWithContext(ctx, func() error {
				return tw.WriteHeader(header)
			}); err != nil {
				closePipeWithError(err)

				return
			}

			if mode.IsRegular() {
				f, err := os.Open(fpath)
				if err != nil {
					log.Warnf("Skipping %s: open failed: %v", fpath, err)

					continue
				}

				err = copyWithContext(ctx, tw, f)

				if e := f.Close(); e != nil {
					log.Warnf("Failed to close file %s: %v", fpath, e)
				}

				if err != nil {
					log.Warnf("Skipping %s: copy failed: %v", fpath, err)

					continue
				}
			}
		}
	}()

	return pr
}

// CreateTarLz4Stream takes a list of files and returns a reader that reads from an lz4-compressed tar archive.
func CreateTarLz4Stream(ctx context.Context, filepaths []string) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		var pipeErr error

		closePipeWithError := func(err error) {
			if pipeErr != nil {
				return
			}

			pipeErr = err
			pw.CloseWithError(err)
		}

		defer func() {
			if pipeErr == nil {
				pw.Close()
			}
		}()

		lw := lz4.NewWriter(pw)

		defer func() {
			if pipeErr == nil {
				if err := lw.Close(); err != nil {
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close lz4 writer: %v", err)
					}
				}
			}
		}()

		tw := tar.NewWriter(lw)

		defer func() {
			if pipeErr == nil {
				if err := tw.Close(); err != nil {
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close tar writer: %v", err)
					}
				}
			}
		}()

		for _, fpath := range filepaths {
			var (
				err        error
				linkTarget string
			)

			select {
			case <-ctx.Done():
				closePipeWithError(ctx.Err())

				return

			default:
			}

			log.Debugf("Adding file: %s", fpath)

			st, err := os.Lstat(fpath)

			if err != nil {
				log.Warnf("Skipping %s: %v", fpath, err)

				continue
			}

			if st.Mode()&os.ModeSymlink != 0 {
				linkTarget, err = os.Readlink(fpath)

				if err != nil {
					log.Warnf("Skipping %s: readlink failed: %v", fpath, err)

					continue
				}
			}

			if st.Mode()&os.ModeSocket != 0 {
				log.Debugf("Skipping socket: %s", fpath)

				continue
			}

			header, err := tar.FileInfoHeader(st, linkTarget)

			if err != nil {
				log.Warnf("Skipping %s: tar header failed: %v", fpath, err)

				continue
			}

			header.Format = tar.FormatGNU
			header.Name = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(fpath)), "/")

			mode := st.Mode()

			if err := writeWithContext(ctx, func() error {
				return tw.WriteHeader(header)
			}); err != nil {
				closePipeWithError(err)

				return
			}

			if mode.IsRegular() {
				f, err := os.Open(fpath)
				if err != nil {
					log.Warnf("Skipping %s: open failed: %v", fpath, err)

					continue
				}

				err = copyWithContext(ctx, tw, f)

				if e := f.Close(); e != nil {
					log.Warnf("Failed to close file %s: %v", fpath, e)
				}

				if err != nil {
					log.Warnf("Skipping %s: copy failed: %v", fpath, err)

					continue
				}
			}
		}
	}()

	return pr
}

// CreateTarXzStream takes a list of files and returns a reader that reads from an xz-compressed tar archive.
func CreateTarXzStream(ctx context.Context, filepaths []string) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		var pipeErr error

		closePipeWithError := func(err error) {
			if pipeErr != nil {
				return
			}

			pipeErr = err
			pw.CloseWithError(err)
		}

		defer func() {
			if pipeErr == nil {
				pw.Close()
			}
		}()

		xw, err := xz.NewWriter(pw)
		if err != nil {
			closePipeWithError(fmt.Errorf("xz writer error: %w", err))

			return
		}

		defer func() {
			if pipeErr == nil {
				if err := xw.Close(); err != nil {
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close xz writer: %v", err)
					}
				}
			}
		}()

		tw := tar.NewWriter(xw)

		defer func() {
			if pipeErr == nil {
				if err := tw.Close(); err != nil {
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close tar writer: %v", err)
					}
				}
			}
		}()

		for _, fpath := range filepaths {
			var (
				err        error
				linkTarget string
			)

			select {
			case <-ctx.Done():
				closePipeWithError(ctx.Err())

				return

			default:
			}

			log.Debugf("Adding file: %s", fpath)

			st, err := os.Lstat(fpath)

			if err != nil {
				log.Warnf("Skipping %s: %v", fpath, err)

				continue
			}

			if st.Mode()&os.ModeSymlink != 0 {
				linkTarget, err = os.Readlink(fpath)

				if err != nil {
					log.Warnf("Skipping %s: readlink failed: %v", fpath, err)

					continue
				}
			}

			if st.Mode()&os.ModeSocket != 0 {
				log.Debugf("Skipping socket: %s", fpath)

				continue
			}

			header, err := tar.FileInfoHeader(st, linkTarget)

			if err != nil {
				log.Warnf("Skipping %s: tar header failed: %v", fpath, err)

				continue
			}

			header.Format = tar.FormatGNU
			header.Name = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(fpath)), "/")

			mode := st.Mode()

			if err := writeWithContext(ctx, func() error {
				return tw.WriteHeader(header)
			}); err != nil {
				closePipeWithError(err)

				return
			}

			if mode.IsRegular() {
				f, err := os.Open(fpath)
				if err != nil {
					log.Warnf("Skipping %s: open failed: %v", fpath, err)

					continue
				}

				err = copyWithContext(ctx, tw, f)

				if e := f.Close(); e != nil {
					log.Warnf("Failed to close file %s: %v", fpath, e)
				}

				if err != nil {
					log.Warnf("Skipping %s: copy failed: %v", fpath, err)

					continue
				}
			}
		}
	}()

	return pr
}

// isPipeClosedError checks if the error is due to a closed pipe.
func isPipeClosedError(err error) bool {
	if err == nil {
		return false
	}

	// Check for the specific pipe closed error message.
	return err.Error() == "io: read/write on closed pipe" ||
		err.Error() == "io: read on closed pipe"
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
