package core

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
)

// TestPreemptableWriter tests preemptableWriter.
func TestPreemptableWriter(t *testing.T) {
	// Set up parameters.
	const (
		checkInterval  = 3
		copyBufferSize = 1024
	)

	// Create a limited input stream. Ensure that it's large enough to trigger a
	// preemption check when copied using our copy buffer.
	source := &io.LimitedReader{
		R: rand.New(rand.NewSource(0)),
		N: (checkInterval + 1) * copyBufferSize,
	}

	// Create a copy buffer.
	copyBuffer := make([]byte, copyBufferSize)

	// Create a closed channel so that the write is already preempted.
	cancelled := make(chan struct{})
	close(cancelled)

	// Create a preemptable writer.
	destinationBuffer := &bytes.Buffer{}
	destination := &preemptableWriter{
		cancelled:     cancelled,
		writer:        destinationBuffer,
		checkInterval: 3,
	}

	// Perform a copy.
	n, err := io.CopyBuffer(destination, source, copyBuffer)

	// Check the copy error.
	if err == nil {
		t.Error("copy not preempted")
	} else if err != errWritePreempted {
		t.Fatal("unexpected copy error:", err)
	}

	// Check the copy sizes.
	if n != int64(destinationBuffer.Len()) {
		t.Error("preemptable writer did not write reported length")
	}
	if n != checkInterval*copyBufferSize {
		t.Error("unexpected number of bytes written between preemption checks")
	}
}
