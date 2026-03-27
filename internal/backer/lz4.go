package backer

import (
	"context"
	"io"

	"github.com/pierrec/lz4"
)

// CreateTarLz4Stream takes a list of files and returns a reader that reads from an lz4-compressed tar archive.
func CreateTarLz4Stream(ctx context.Context, filepaths []string) io.ReadCloser {
	return createArchiveStream(ctx, filepaths, func(w io.Writer) (io.WriteCloser, error) {
		return lz4.NewWriter(w), nil
	})
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
