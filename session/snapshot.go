package session

import (
	"github.com/gogo/protobuf/proto"

	"github.com/havoc-io/mutagen/sync"
)

// marshalEntry marshals an Entry message inside of an Archive, allowing for
// representation of nil Entry objects. It should be used in conjunction with
// unmarshalEntry.
func marshalEntry(entry *sync.Entry) ([]byte, error) {
	return proto.Marshal(&Archive{Root: entry})
}

// unmarshalEntry unmarshals an Entry message from inside of an Archive,
// allowing for representation of nil Entry objects. It should be used in
// conjunction with marshalEntry.
func unmarshalEntry(encoded []byte) (*sync.Entry, error) {
	archive := &Archive{}
	if err := proto.Unmarshal(encoded, archive); err != nil {
		return nil, err
	}
	return archive.Root, nil
}
