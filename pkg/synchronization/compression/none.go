package compression

import (
	"io"

	"github.com/mutagen-io/mutagen/pkg/stream"
)

// noneCompressor implements stream.WriteFlushCloser for uncompressed streams.
type noneCompressor struct {
	// Writer is the underlying writer.
	io.Writer
}

// Flush implements stream.Flusher.Flush.
func (c *noneCompressor) Flush() error {
	return nil
}

// Close implements io.Closer.Close.
func (c *noneCompressor) Close() error {
	return nil
}

// compressNone implements no-op compression for uncompressed streams.
func compressNone(compressed io.Writer) stream.WriteFlushCloser {
	return &noneCompressor{compressed}
}

// decompressNone implements no-op decompression for uncompressed streams.
func decompressNone(compressed io.Reader) io.ReadCloser {
	return io.NopCloser(compressed)
}
