package message

import (
	"bytes"
	"io"
	"testing"
)

type sampleMessage struct {
	Message string
	Count   int
}

func runStreamClean(t *testing.T, compressed bool) {
	// Mark this as a helper function.
	t.Helper()

	// Create a byte buffer to use as a transport.
	buffer := &bytes.Buffer{}

	// Wrap it in a message stream.
	stream := NewStream(buffer, compressed)

	// Write a message to the stream.
	if err := stream.Encode(sampleMessage{"content", 100}); err != nil {
		t.Fatal("unable to encode sample message:", err)
	}

	// Read it from the stream.
	var message sampleMessage
	if err := stream.Decode(&message); err != nil {
		t.Fatal("unable to decode sample message:", err)
	}

	// Verify that the transport is left clean.
	if buffer.Len() != 0 {
		t.Error("transport left dirty")
	}
}

func TestStreamClean(t *testing.T) {
	runStreamClean(t, false)
}

func TestCompressedStreamClean(t *testing.T) {
	runStreamClean(t, true)
}

type badReadWriter struct{}

func (r *badReadWriter) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (w *badReadWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestEOFDecode(t *testing.T) {
	// Create a bad connection.
	connection := &badReadWriter{}

	// Create a message stream.
	stream := NewStream(connection, false)

	// Ensure that the decode fails and returns the EOF error unwrapped.
	var message sampleMessage
	if err := stream.Decode(&message); err != io.EOF {
		t.Error("expected EOF from connection at EOF")
	}
}

func TestEncodeFail(t *testing.T) {
	// Create a bad connection.
	connection := &badReadWriter{}

	// Create a message stream without compression so that we can check for
	// failures encoding to the underlying connection.
	stream := NewStream(connection, false)

	// Ensure that the encode fails and returns the ErrClosedPipe error
	// unwrapped.
	var message sampleMessage
	if err := stream.Encode(&message); err != io.ErrClosedPipe {
		t.Error("expected ErrClosedPipe from closed connection")
	}
}

func TestEncodeFailOnFlush(t *testing.T) {
	// Create a bad connection.
	connection := &badReadWriter{}

	// Create a message stream with compression so that we can check for
	// failures flushing to the underlying connection. This assumes that the
	// compressors buffer is large enough not to flush during the encode.
	stream := NewStream(connection, true)

	// Ensure that the encode fails and returns the ErrClosedPipe error
	// unwrapped.
	var message sampleMessage
	if err := stream.Encode(&message); err != io.ErrClosedPipe {
		t.Error("expected ErrClosedPipe from closed connection")
	}
}
