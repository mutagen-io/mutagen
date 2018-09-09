package compression

import (
	"compress/flate"
	"io"

	"github.com/pkg/errors"
)

const (
	// defaultCompressionLevel is the default compression level to use for
	// writers.
	defaultCompressionLevel = 6
)

// NewDecompressingReader wraps an io.Reader in a decompressor.
func NewDecompressingReader(source io.Reader) io.Reader {
	// HACK: Technically this function returns an io.ReadCloser, and it
	// documents that it is the caller's function to call Close on the reader.
	// However, it turns out that the underlying implementation of Close just
	// checks for stream errors and that it isn't really necessary to call it.
	// As a result, we ignore this portion of the interface, but we're sort of
	// relying on an implementation detail in doing so.
	return flate.NewReader(source)
}

// automaticallyFlushingFlateWriter is a type that wraps a flate.Writer and
// automatically flushes data on every write.
type automaticallyFlushingFlateWriter struct {
	// compressor is the underlying flate compressor.
	compressor *flate.Writer
}

// Write writes data to the compressor and automatically flushes it to the
// underlying writer.
func (w *automaticallyFlushingFlateWriter) Write(buffer []byte) (int, error) {
	count, err := w.compressor.Write(buffer)
	if err != nil {
		return count, err
	} else if err = w.compressor.Flush(); err != nil {
		return 0, errors.Wrap(err, "unable to flush compressor")
	}
	return count, nil
}

// NewCompressingWriter wraps an io.Writer in a compressor.
func NewCompressingWriter(destination io.Writer) io.Writer {
	// Create the compressor. If a sane compression level is provided, the flate
	// API guarantees that creation of the compressor will succeed.
	compressor, _ := flate.NewWriter(destination, defaultCompressionLevel)

	// Wrap the compressor.
	return &automaticallyFlushingFlateWriter{compressor}
}
