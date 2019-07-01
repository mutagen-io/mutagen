package mutagen

import (
	"bytes"
	"io"
	"testing"
)

// receiveAndCompareVersion is a test helper function that reads version
// information from the specified reader and ensures that it matches the current
// version. Version tag components are neither transmitted nor received, so they
// do not enter into this comparison.
func receiveAndCompareVersion(reader io.Reader) (bool, error) {
	// Receive the version.
	major, minor, patch, err := receiveVersion(reader)
	if err != nil {
		return false, err
	}

	// Compare the version.
	return major == VersionMajor &&
		minor == VersionMinor &&
		patch == VersionPatch, nil
}

// TestVersionSendReceiveAndCompare tests a version send/receive cycle.
func TestVersionSendReceiveAndCompare(t *testing.T) {
	// Create an intermediate buffer.
	buffer := &bytes.Buffer{}

	// Send the version.
	if err := sendVersion(buffer); err != nil {
		t.Fatal("unable to send version:", err)
	}

	// Ensure that the buffer is non-empty.
	if buffer.Len() != 12 {
		t.Fatal("buffer does not contain expected byte count")
	}

	// Receive the version.
	if match, err := receiveAndCompareVersion(buffer); err != nil {
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
	match, err := receiveAndCompareVersion(buffer)
	if err == nil {
		t.Error("version received from empty buffer")
	}
	if match {
		t.Error("version match on empty buffer")
	}
}

// TODO: Add tests for ClientVersionHandshake and ServerVersionHandshake.
