package backer

import (
	"context"
	"io"

	"github.com/pierrec/lz4"
)

// CreateTarLz4Stream takes a list of files and returns a reader that reads from an lz4-compressed tar archive.
func CreateTarLz4Stream(ctx context.Context, filepaths []string) io.ReadCloser {
	return createArchiveStream(ctx, filepaths, func(w io.Writer) (io.WriteCloser, error) {
		lz4Writer := lz4.NewWriter(w)
		// Map compression_level (1-9) to lz4 compression level.
		// 1-3: fast, 4-6: balanced, 7-9: better compression.
		lz4Writer.Header.CompressionLevel = mapCompressionLevelToLz4(C.CompressionLevel) //nolint: staticcheck

		return lz4Writer, nil
	})
}

// mapCompressionLevelToLz4 maps 1-9 compression level to lz4 compression level.
// 1-3: fast, 4-6: balanced, 7-9: better compression.
func mapCompressionLevelToLz4(level int) int {
	switch {
	case level <= 3:
		return 0 // Fastest.
	case level <= 6:
		return 1 // Default.
	default:
		return 2 // Best compression.
	}
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
