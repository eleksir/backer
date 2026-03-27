package backer

import (
	"context"
	"io"

	"github.com/dsnet/compress/bzip2"
)

// CreateTarBzip2Stream takes a list of files and returns a reader that reads from a bzip2-compressed tar archive.
func CreateTarBzip2Stream(ctx context.Context, filepaths []string) io.ReadCloser {
	return createArchiveStream(ctx, filepaths, func(w io.Writer) (io.WriteCloser, error) {
		return bzip2.NewWriter(w, nil)
	})
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
