package backer

import (
	"context"
	"io"

	"github.com/klauspost/compress/zstd"
)

// CreateTarZstdStream takes a list of files and returns a reader that reads from a zstd-compressed tar archive.
func CreateTarZstdStream(ctx context.Context, filepaths []string) io.ReadCloser {
	return createArchiveStream(ctx, filepaths, func(w io.Writer) (io.WriteCloser, error) {
		return zstd.NewWriter(w, zstd.WithEncoderLevel(zstdLevel(C.CompressionLevel)))
	})
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

/* vim: setlocal ft=go noet ai ts=4 sw=4 sts=4: */
