package core

import (
	"errors"
	"io"
)

var (
	// errWritePreempted indicates that a write was preempted in a
	// preemptableWriter instance.
	errWritePreempted = errors.New("write preempted")
)

// preemptableWriter is an io.Writer implementation that checks for preemption
// every N writes.
type preemptableWriter struct {
	// cancelled is the channel that, when closed, indicates preemption.
	cancelled <-chan struct{}
	// writer is the underlying writer.
	writer io.Writer
	// checkInterval is the number of writes to allow between preemption checks.
	checkInterval uint
	// writeCount is the number of writes since the last preemption check.
	writeCount uint
}

// Write implements io.Writer.Write.
func (w *preemptableWriter) Write(data []byte) (int, error) {
	// Handle preemption checking.
	if w.writeCount >= w.checkInterval {
		select {
		case <-w.cancelled:
			return 0, errWritePreempted
		default:
		}
		w.writeCount = 0
	} else {
		w.writeCount++
	}

	// Perform the write.
	return w.writer.Write(data)
}
