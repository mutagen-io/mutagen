package compression

import (
	"io"

	"github.com/klauspost/compress/flate"

	"github.com/mutagen-io/mutagen/pkg/stream"
)

// compressDeflate implements compression for DEFLATE streams.
func compressDeflate(compressed io.Writer) stream.WriteFlushCloser {
	// Create the compressor. We check for errors, but we don't include them as
	// part of the interface because they can only occur with an invalid
	// compressor configuration (which can't occur when we only use defaults).
	compressor, err := flate.NewWriter(compressed, flate.DefaultCompression)
	if err != nil {
		panic("DEFLATE compressor construction failed")
	}

	// Success.
	return compressor
}

// decompressDeflate implements decompression for DEFLATE streams.
func decompressDeflate(compressed io.Reader) io.ReadCloser {
	return flate.NewReader(compressed)
}
