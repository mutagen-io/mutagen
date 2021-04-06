package stream

import (
	"errors"
	"io"
)

var (
	// ErrWritePreempted indicates that a write operation was preempted.
	ErrWritePreempted = errors.New("write preempted")
)

// preemptableWriter is an io.Writer implementation that checks for preemption
// every N writes.
type preemptableWriter struct {
	// writer is the underlying writer.
	writer io.Writer
	// cancelled is the channel that, when closed, indicates preemption.
	cancelled <-chan struct{}
	// checkInterval is the number of writes to allow between preemption checks.
	checkInterval uint
	// writeCount is the number of writes since the last preemption check.
	writeCount uint
}

// NewPreemptableWriter wraps an io.Writer and provides preemption capabilities
// for long copy operations. It takes an underlying writer, a channel that (once
// closed) indicates cancellation, and an interval that specifies the maximum
// number of Write calls that should be processed between cancellation checks.
// If interval is 0, a cancellation check will be performed before every write.
func NewPreemptableWriter(writer io.Writer, cancelled <-chan struct{}, interval uint) io.Writer {
	return &preemptableWriter{
		writer:        writer,
		cancelled:     cancelled,
		checkInterval: interval,
	}
}

// Write implements io.Writer.Write.
func (w *preemptableWriter) Write(data []byte) (int, error) {
	// Handle preemption checking.
	if w.writeCount == w.checkInterval {
		select {
		case <-w.cancelled:
			return 0, ErrWritePreempted
		default:
		}
		w.writeCount = 0
	} else {
		w.writeCount++
	}

	// Perform the write.
	return w.writer.Write(data)
}
