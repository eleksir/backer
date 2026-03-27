// Package backer implements HTTP/HTTPS backup server.
package backer

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"backer/internal/log"
)

// createArchiveStream creates a pipe and streams a tar archive compressed by the provided writer.
// The writerFactory creates the compression writer wrapping the pipe writer.
// This is the core of all CreateTar*Stream functions.
func createArchiveStream(ctx context.Context, filepaths []string, writerFactory func(io.Writer) (io.WriteCloser, error)) io.ReadCloser {
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

		cw, err := writerFactory(pw)
		if err != nil {
			closePipeWithError(fmt.Errorf("compression writer error: %w", err))

			return
		}

		defer func() {
			if pipeErr == nil {
				if err := cw.Close(); err != nil {
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close compression writer: %v", err)
					}
				}
			}
		}()

		tw := tar.NewWriter(cw)

		defer func() {
			if pipeErr == nil {
				if err := tw.Close(); err != nil {
					if !isPipeClosedError(err) {
						log.Errorf("Failed to close tar writer: %v", err)
					}
				}
			}
		}()

		writeFilesToTar(ctx, filepaths, tw, closePipeWithError)
	}()

	return pr
}

// writeFilesToTar iterates over filepaths and writes each file to the tar archive.
func writeFilesToTar(ctx context.Context, filepaths []string, tw *tar.Writer, closePipeWithError func(error)) {
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
