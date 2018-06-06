package sync

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
)

func TestArchiveConsistentSerialization(t *testing.T) {
	// Create two entries. Although not strictly neccessary, make them distinct
	// values.
	firstEntry := testDirectory1Entry
	secondEntry := firstEntry.Copy()

	// Serialize the first entry.
	firstBuffer := proto.NewBuffer(nil)
	firstBuffer.SetDeterministic(true)
	if err := firstBuffer.Marshal(&Archive{Root: firstEntry}); err != nil {
		t.Fatal("unable to marshal first entry:", err)
	}

	// Serialize the second entry.
	secondBuffer := proto.NewBuffer(nil)
	secondBuffer.SetDeterministic(true)
	if err := secondBuffer.Marshal(&Archive{Root: secondEntry}); err != nil {
		t.Fatal("unable to marshal second entry:", err)
	}

	// Ensure that they're equal.
	if !bytes.Equal(firstBuffer.Bytes(), secondBuffer.Bytes()) {
		t.Error("marshalling is not consistent")
	}
}
