package stream

import (
	"io"
	"sync"
)

// concurrentWriter is an io.Writer that serializes calls to Write.
type concurrentWriter struct {
	// lock serializes operations on the writer.
	lock sync.Mutex
	// writer is the underlying writer.
	writer io.Writer
}

// NewConcurrentWriter creates a new writer that serializes operations on the
// underlying writer.
func NewConcurrentWriter(writer io.Writer) io.Writer {
	return &concurrentWriter{writer: writer}
}

// Write implements io.Writer.Write.
func (w *concurrentWriter) Write(buffer []byte) (int, error) {
	// Lock the writer and defer its release.
	w.lock.Lock()
	defer w.lock.Unlock()

	// Perform the write.
	return w.writer.Write(buffer)
}
