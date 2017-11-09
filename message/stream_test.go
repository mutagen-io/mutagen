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

func TestStreamClean(t *testing.T) {
	// Create a byte buffer to use as a transport.
	buffer := &bytes.Buffer{}

	// Wrap it in a message stream.
	stream := NewStream(buffer)

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
	stream := NewStream(connection)

	// Ensure that the decode fails and returns the EOF error unwrapped.
	var message sampleMessage
	if err := stream.Decode(&message); err != io.EOF {
		t.Error("expected EOF from connection at EOF")
	}
}

func TestEncodeFail(t *testing.T) {
	// Create a bad connection.
	connection := &badReadWriter{}

	// Create a message stream.
	stream := NewStream(connection)

	// Ensure that the encode fails and returns the ErrClosedPipe error
	// unwrapped.
	var message sampleMessage
	if err := stream.Encode(&message); err != io.ErrClosedPipe {
		t.Error("expected ErrClosedPipe from closed connection")
	}
}
