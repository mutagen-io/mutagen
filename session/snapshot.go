package session

import (
	"crypto/sha1"

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

// checksum computes the checksum of a serialized entry. The checksum that's
// used is not stable and should only be used for transfer verification within a
// synchronization cycle between the daemon and agent. Its result should never
// be persisted anywhere (e.g. disk) that would require compatibility in future
// versions.
func checksum(snapshotBytes []byte) []byte {
	result := sha1.Sum(snapshotBytes)
	return result[:]
}
