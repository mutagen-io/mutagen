package mutagen

import (
	"bytes"
	"testing"
)

// TestVersionSendReceiveAndCompare tests a version send/receive cycle.
func TestVersionSendReceiveAndCompare(t *testing.T) {
	// Create an intermediate buffer.
	buffer := &bytes.Buffer{}

	// Send the version.
	if err := SendVersion(buffer); err != nil {
		t.Fatal("unable to send version:", err)
	}

	// Ensure that the buffer is non-empty.
	if buffer.Len() != 12 {
		t.Fatal("buffer does not contain expected byte count")
	}

	// Receive the version.
	if match, err := ReceiveAndCompareVersion(buffer); err != nil {
		t.Fatal("unable to receive version:", err)
	} else if !match {
		t.Error("version mismatch on receive")
	}
}

// TestVersionReceiveAndCompareEmptyBuffer tests that receiving a version fails
// when reading from an empty buffer.
func TestVersionReceiveAndCompareEmptyBuffer(t *testing.T) {
	// Create an empty buffer.
	buffer := &bytes.Buffer{}

	// Attempt to receive the version.
	match, err := ReceiveAndCompareVersion(buffer)
	if err == nil {
		t.Error("version received from empty buffer")
	}
	if match {
		t.Error("version match on empty buffer")
	}
}
