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

// Flusher represents a stream that performs internal buffering that may need to
// be flushed to ensure transmission.
type Flusher interface {
	// Flush forces transmission of any buffered stream data.
	Flush() error
}

// WriteFlushCloser represents a stream with writing, flushing, and closing
// functionality.
type WriteFlushCloser interface {
	io.Writer
	Flusher
	io.Closer
}
