package frame

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/havoc-io/mutagen/sync"
)

// HACK: For our tests, we just use a message definition from another package.
// It'll be a pain to generate *.pb.go files in this package that only get built
// for tests, and the sync package isn't going anywhere. Plus, sync.Entry
// provides equality testing.

// testFraming tests that the provided entry successfully encodes and decodes.
func testFraming(t *testing.T, entry *sync.Entry) {
	// Create a transport.
	transport := &bytes.Buffer{}

	// Create an encoder and encode the message.
	encoder := NewEncoder(transport)
	if err := encoder.Encode(entry); err != nil {
		t.Fatal("unable to encode message:", err)
	}

	// Create a decoder and decode the message.
	decoder := NewDecoder(transport)
	decoded := &sync.Entry{}
	if err := decoder.DecodeTo(decoded); err != nil {
		t.Fatal("unable to decode message:", err)
	}

	// Verify equality.
	if !decoded.Equal(entry) {
		t.Error("decoded message does not match original")
	}

	// Verify that the transport is clean.
	if transport.Len() > 0 {
		t.Error("framing did not leave transport clean")
	}
}

func TestFramingReusable(t *testing.T) {
	// Create a small message that should fit into the reusable buffers.
	entry := &sync.Entry{
		Kind:       sync.EntryKind_File,
		Executable: true,
		Digest:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	// Test.
	testFraming(t, entry)
}

func TestFramingNonReusable(t *testing.T) {
	// Create a large message that will require buffer allocation.
	entry := &sync.Entry{
		Kind:       sync.EntryKind_File,
		Executable: true,
		Digest:     make([]byte, 2*reusableBufferSize),
	}

	// Test.
	testFraming(t, entry)
}

func TestFramingTooLarge(t *testing.T) {
	// Create a message that will be too large to frame.
	entry := &sync.Entry{
		Digest: make([]byte, 2*maximumMessageSize),
	}

	// Create a transport.
	transport := &bytes.Buffer{}

	// Create an encoder and try to encode the message.
	encoder := NewEncoder(transport)
	if encoder.Encode(entry) == nil {
		t.Fatal("encoding of message too large for framing should fail")
	}
}

func TestDecodingTooLarge(t *testing.T) {
	// Create a transport.
	transport := &bytes.Buffer{}

	// Encode a length header that's too big to decode.
	var bigSizeBytes [maximumMessageUvarintLength + 1]byte
	headerSize := binary.PutUvarint(bigSizeBytes[:], maximumMessageSize+1)
	transport.Write(bigSizeBytes[:headerSize])

	// Create a decoder and try to decode.
	// TODO: We should probably use a sentinel error to verify that the failure
	// is due to the decoder rejecting the size.
	decoder := NewDecoder(transport)
	decoded := &sync.Entry{}
	if decoder.DecodeTo(decoded) == nil {
		t.Fatal("decoding of message too large for framing should fail")
	}
}
