package backer

import (
	"compress/gzip"
	"context"
	"io"

	"github.com/klauspost/pgzip"
)

// CreateTarGzStream takes a list of files and returns a reader that reads from a gzip-compressed tar archive.
func CreateTarGzStream(ctx context.Context, filepaths []string) io.ReadCloser {
	return createArchiveStream(ctx, filepaths, func(w io.Writer) (io.WriteCloser, error) {
		return gzip.NewWriterLevel(w, C.CompressionLevel)
	})
}

// CreateTarPgzipStream takes a list of files and returns a reader that reads from a parallel gzip-compressed tar archive.
func CreateTarPgzipStream(ctx context.Context, filepaths []string) io.ReadCloser {
	return createArchiveStream(ctx, filepaths, func(w io.Writer) (io.WriteCloser, error) {
		return pgzip.NewWriterLevel(w, C.CompressionLevel)
	})
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
