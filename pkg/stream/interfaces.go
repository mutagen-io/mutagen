package stream

import (
	"io"
)

// DualModeReader represents a reader that can perform both regular and
// single-byte reads efficiently.
type DualModeReader interface {
	io.ByteReader
	io.Reader
}

// CloseWriter represents a stream with half-closure functionality.
type CloseWriter interface {
	io.Writer
	// CloseWrite closes the stream for writes and signals io.EOF to the
	// receiving end of the stream. It must unblock any pending calls to Write.
	CloseWrite() error
}
