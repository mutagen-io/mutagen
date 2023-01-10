package stream

import (
	"io"
)

// flushCloser is the io.Closer implementation underlying NewFlushCloser.
type flushCloser struct {
	// flusher is the underlying flusher.
	flusher Flusher
}

// NewFlushCloser creates a new io.Closer that aliases Close to the specified
// flusher's Flush method. It is primarily used for bufio.Writer instances.
func NewFlushCloser(flusher Flusher) io.Closer {
	return &flushCloser{flusher}
}

// Close implements io.Closer.Close.
func (c *flushCloser) Close() error {
	return c.flusher.Flush()
}
