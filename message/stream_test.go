package message

import (
	"bytes"
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
