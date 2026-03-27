package backer

import (
	"context"
	"io"

	"github.com/ulikunitz/xz"
)

// CreateTarXzStream takes a list of files and returns a reader that reads from an xz-compressed tar archive.
func CreateTarXzStream(ctx context.Context, filepaths []string) io.ReadCloser {
	return createArchiveStream(ctx, filepaths, func(w io.Writer) (io.WriteCloser, error) {
		return xz.NewWriter(w)
	})
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
