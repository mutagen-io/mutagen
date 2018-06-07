package sync

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
)

func TestArchiveEmptyDifferentEmptyDirectory(t *testing.T) {
	// Serialize an archive with a nil root. It should be an empty byte
	// sequence.
	emptyArchive := &Archive{}
	emptyArchiveBytes, err := proto.Marshal(emptyArchive)
	if err != nil {
		t.Fatal("unable to marshal empty archive:", err)
	}
	if len(emptyArchiveBytes) > 0 {
		t.Error("empty archive serialized to non-empty bytes")
	}

	// Serialize an archive with an empty directory at the root.
	archiveEmptyDirectory := &Archive{Root:&Entry{Kind: EntryKind_Directory}}
	archiveEmptyDirectoryBytes, err := proto.Marshal(archiveEmptyDirectory)
	if err != nil {
		t.Fatal("unable to marshal archive with empty directory:", err)
	}

	// Ensure they differ.
	if bytes.Equal(emptyArchiveBytes, archiveEmptyDirectoryBytes) {
		t.Error("empty archive and archive with empty directory serialize the same")
	}
}

func TestArchiveConsistentSerialization(t *testing.T) {
	// Create two entries. Although not strictly necessary, make them distinct
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
