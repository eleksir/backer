package backer

import (
	"context"
	"io"

	"github.com/dsnet/compress/bzip2"
)

// CreateTarBzip2Stream takes a list of files and returns a reader that reads from a bzip2-compressed tar archive.
func CreateTarBzip2Stream(ctx context.Context, filepaths []string) io.ReadCloser {
	return createArchiveStream(ctx, filepaths, func(w io.Writer) (io.WriteCloser, error) {
		level := mapCompressionLevelToBzip2(C.CompressionLevel)

		return bzip2.NewWriter(w, &bzip2.WriterConfig{Level: level})
	})
}

// mapCompressionLevelToBzip2 maps 1-9 compression level to bzip2 level.
func mapCompressionLevelToBzip2(level int) int {
	if level < 1 {
		return bzip2.DefaultCompression
	}
	if level > 9 {
		return bzip2.BestCompression
	}

	return level
}

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
