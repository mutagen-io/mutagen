package webrtcutil

import (
	"io"
	"testing"
	"time"
)

// TestPipeWriteUnblockOnCloseWithError tests that io.PipeWriter.Write unblocks
// when the pipe is closed from the write end using CloseWithError. This is
// technically an implementation detail, so we use this test to ensure that we
// can rely on this behavior.
func TestPipeWriteUnblockOnCloseWithError(t *testing.T) {
	// Create a pipe.
	reader, writer := io.Pipe()

	// Perform a write and monitor for its completion.
	done := make(chan struct{})
	go func() {
		writer.Write([]byte{0, 0})
		close(done)
	}()

	// Read at least one byte to ensure the write has started.
	var output [1]byte
	if _, err := reader.Read(output[:]); err != nil {
		t.Fatal("unable to read initial byte:", err)
	}

	// Preempt the write. We use CloseWithError because that's the behavior we
	// care about.
	writer.CloseWithError(nil)

	// Wait for the writer to terminate.
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("write preemption timeout")
	}
}

// TODO: Implement additional tests.
