package stream

import (
	"io"
	"sync"
)

// ValveWriter is an io.Writer that wraps another io.Writer and forwards writes
// to it until the ValveWriter's internal valve is shut, after which writes will
// continue to succeed but not actually be written to the underlying writer.
type ValveWriter struct {
	// writerLock serializes access to the underlying writer.
	writerLock sync.Mutex
	// writer is the underlying writer.
	writer io.Writer
}

// NewValveWriter creates a new ValveWriter instance using the specified writer.
// The writer may be nil, in which case the writer will start pre-shut.
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

// Shut closes the valve and prevents future writes to the underlying writer. It
// is safe to call Shut concurrently with Write, but doing so will not preempt
// or unblock pending calls to Write.
func (w *ValveWriter) Shut() {
	// Lock the writer and defer its release.
	w.writerLock.Lock()
	defer w.writerLock.Unlock()

	// Nil out the writer to stop any future writes to it.
	w.writer = nil
}
