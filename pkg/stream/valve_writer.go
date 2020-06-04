package stream

import (
	"io"
	"sync"
)

// ValveWriter is an io.Writer that wraps another io.Writer and performs writes
// to it until the valve is shut, after which writes will continue to succeed
// but not be written to the underlying writer.
type ValveWriter struct {
	// writerLock serializes access to the underlying writer.
	writerLock sync.Mutex
	// writer is the underlying writer.
	writer io.Writer
}

// NewValveWriter creates a new ValveWriter instance using the specified writer.
// The write may be nil, in which case the writer will start pre-shut.
func NewValveWriter(writer io.Writer) *ValveWriter {
	return &ValveWriter{writer: writer}
}

// Write implements io.Writer.Write.
func (w *ValveWriter) Write(buffer []byte) (int, error) {
	// Lock the writer and defer its release.
	w.writerLock.Lock()
	defer w.writerLock.Unlock()

	// If there's no writer, then just pretend that we wrote all of the data.
	if w.writer == nil {
		return len(buffer), nil
	}

	// Otherwise write to the underlying writer.
	return w.writer.Write(buffer)
}

// Shut closes the valve and stops writes to the underlying writer.
func (w *ValveWriter) Shut() {
	// Lock the writer and defer its release.
	w.writerLock.Lock()
	defer w.writerLock.Unlock()

	// Nil out the writer to stop any future writes to it.
	w.writer = nil
}
