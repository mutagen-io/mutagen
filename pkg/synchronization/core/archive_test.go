package core

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
)

// TestArchiveEnsureValid tests Archive.EnsureValid.
func TestArchiveEnsureValid(t *testing.T) {
	// Ensure that a nil archive is considered invalid in all cases.
	var archive *Archive
	if archive.EnsureValid(false) == nil {
		t.Error("nil archive incorrectly classified as valid (without synchronizability requirement)")
	}
	if archive.EnsureValid(true) == nil {
		t.Error("nil archive incorrectly classified as valid (when requiring synchronizability)")
	}

	// Process test cases.
	for i, test := range entryEnsureValidTestCases {
		// Compute a description for the test in case we need it.
		description := "without synchronizability requirement"
		if test.synchronizable {
			description = "when requiring synchronizability"
		}

		// Check validity.
		archive := &Archive{Content: test.entry}
		err := archive.EnsureValid(test.synchronizable)
		valid := err == nil
		if valid != test.expected {
			if valid {
				t.Errorf("test index %d: archive incorrectly classified as valid (%s)", i, description)
			} else {
				t.Errorf("test index %d: archive incorrectly classified as invalid (%s): %v", i, description, err)
			}
		}
	}
}

// TestArchiveNilEmptyContentDistinction tests that archive serialization can
// distinguish between a nil root entry and an empty root directory.
func TestArchiveNilEmptyContentDistinction(t *testing.T) {
	// Serialize an archive with a nil root and verify that it encodes to an
	// empty buffer.
	nilContentArchive := &Archive{}
	nilArchiveBytes, err := proto.Marshal(nilContentArchive)
	if err != nil {
		t.Fatal("unable to marshal archive with nil root entry:", err)
	}
	if len(nilArchiveBytes) > 0 {
		t.Error("archive with nil root entry serialized to non-empty bytes")
	}

	// Serialize an archive with an empty directory at the root.
	emptyContentArchive := &Archive{Content: &Entry{}}
	emptyContentArchiveBytes, err := proto.Marshal(emptyContentArchive)
	if err != nil {
		t.Fatal("unable to marshal archive with empty root directory:", err)
	}

	// Ensure that the results differ.
	if bytes.Equal(nilArchiveBytes, emptyContentArchiveBytes) {
		t.Error("archive with nil root and archive with empty root serialize the same")
	}
}
