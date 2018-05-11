package session

import (
	"github.com/golang/protobuf/proto"

	"github.com/havoc-io/mutagen/sync"
)

// marshalEntry marshals an Entry message inside of an Archive with optional
// deterministic serialization. This allows for representation of nil Entry
// objects. It should be used in conjunction with unmarshalEntry.
func marshalEntry(entry *sync.Entry, deterministic bool) ([]byte, error) {
	// Create a buffer in which to serialize.
	buffer := proto.NewBuffer(nil)

	// Set deterministic serialization behavior.
	buffer.SetDeterministic(deterministic)

	// Marhsal into the buffer.
	if err := buffer.Marshal(&Archive{Root: entry}); err != nil {
		return nil, err
	}

	// Done.
	return buffer.Bytes(), nil
}

// unmarshalEntry unmarshals an Entry message from inside of an Archive,
// allowing for representation of nil Entry objects. It should be used in
// conjunction with marshalEntry.
func unmarshalEntry(encoded []byte) (*sync.Entry, error) {
	// Allocate an empty archive in which to unmarshal.
	archive := &Archive{}

	// Perform unmarshalling.
	if err := proto.Unmarshal(encoded, archive); err != nil {
		return nil, err
	}

	// Done.
	return archive.Root, nil
}
