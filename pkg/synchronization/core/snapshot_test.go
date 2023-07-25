package core

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
)

// TestSnapshotEnsureValid tests Snapshot.EnsureValid.
func TestSnapshotEnsureValid(t *testing.T) {
	// Ensure that a nil snapshot is considered invalid in all cases.
	var snapshot *Snapshot
	if snapshot.EnsureValid() == nil {
		t.Error("nil snapshot incorrectly classified as valid")
	}

	// Process test cases. We skip test cases where unsynchronizable content is
	// disallowed because snapshots allow all unsynchronizable content.
	for i, test := range entryEnsureValidTestCases {
		if test.synchronizable {
			continue
		}
		snapshot := &Snapshot{Content: test.entry}
		err := snapshot.EnsureValid()
		valid := err == nil
		if valid != test.expected {
			if valid {
				t.Errorf("test index %d: snapshot incorrectly classified as valid", i)
			} else {
				t.Errorf("test index %d: snapshot incorrectly classified as invalid: %v", i, err)
			}
		}
	}
}

// TestSnapshotNilEmptyContentDistinction tests that snapshot serialization can
// distinguish between a nil root entry and an empty root directory.
func TestSnapshotNilEmptyContentDistinction(t *testing.T) {
	// Serialize a snapshot with a nil root and verify that it encodes to an
	// empty buffer.
	nilContentSnapshot := &Snapshot{}
	nilSnapshotBytes, err := proto.Marshal(nilContentSnapshot)
	if err != nil {
		t.Fatal("unable to marshal snapshot with nil root entry:", err)
	}
	if len(nilSnapshotBytes) > 0 {
		t.Error("snapshot with nil root entry serialized to non-empty bytes")
	}

	// Serialize a snapshot with an empty directory at the root.
	emptyContentSnapshot := &Snapshot{Content: &Entry{}}
	emptyContentSnapshotBytes, err := proto.Marshal(emptyContentSnapshot)
	if err != nil {
		t.Fatal("unable to marshal snapshot with empty root directory:", err)
	}

	// Ensure that the results differ.
	if bytes.Equal(nilSnapshotBytes, emptyContentSnapshotBytes) {
		t.Error("snapshot with nil root and snapshot with empty root serialize the same")
	}
}

// TestSnapshotConsistentSerialization tests that Protocol Buffers'
// deterministic marshalling behaves correctly with Snapshot. This is really a
// test of Protocol Buffers' implementation, but it's so performance-critical
// for us that it warrants a test to serve as a canary.
func TestSnapshotConsistentSerialization(t *testing.T) {
	// Create two entries, one of which is a deep copy of the other. We could
	// also just serialize the same entry twice, but this is more rigorous.
	firstEntry := tDM
	secondEntry := firstEntry.Copy(EntryCopyBehaviorDeep)

	// Configure Protocol Buffers marshaling to be deterministic.
	marshaling := proto.MarshalOptions{Deterministic: true}

	// Serialize the first snapshot.
	firstBytes, err := marshaling.Marshal(&Snapshot{
		Content:                firstEntry,
		PreservesExecutability: true,
	})
	if err != nil {
		t.Fatal("unable to marshal first snapshot:", err)
	}

	// Serialize the second snapshot.
	secondBytes, err := marshaling.Marshal(&Snapshot{
		Content:                secondEntry,
		PreservesExecutability: true,
	})
	if err != nil {
		t.Fatal("unable to marshal second snapshot:", err)
	}

	// Ensure that they're equal.
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Error("snapshot marshalling is not consistent")
	}
}
