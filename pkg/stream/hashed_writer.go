package stream

import (
	"hash"
	"io"
)

// hashedWriter is the io.Writer implementation underlying NewHashedWriter.
type hashedWriter struct {
	// writer is the underlying writer.
	writer io.Writer
	// hasher is the associated hash function.
	hasher hash.Hash
}

// NewHashedWriter creates a new io.Writer that attaches a hash function to an
// existing writer, ensuring that the hash processes all bytes that are
// successfully written to the associated writer.
func NewHashedWriter(writer io.Writer, hasher hash.Hash) io.Writer {
	return &hashedWriter{writer, hasher}
}

// Write implements io.Writer.Write.
func (w *hashedWriter) Write(data []byte) (int, error) {
	// Write to the underlying writer.
	n, err := w.writer.Write(data)

	// Write the corresponding bytes to the hasher. This write can't fail, so we
	// can safely assume that all provided bytes are processed.
	w.hasher.Write(data[:n])

	// Done.
	return n, err
}
