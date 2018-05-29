package sync

import (
	"testing"
)

func TestExecutabilityStripNil(t *testing.T) {
	if StripExecutability(nil) != nil {
		t.Fatal("executability stripping of nil entry did not return nil")
	}
}

func TestExecutabilityPropagateNil(t *testing.T) {
	if PropagateExecutability(testDirectoryEntry, nil) != nil {
		t.Fatal("executability propagation to nil entry did not return nil")
	}
}

func TestExecutabilityPropagationCycle(t *testing.T) {
	// Create a copy of the test directory entry with executability stripped and
	// ensure that it differs.
	stripped := StripExecutability(testDirectoryEntry)
	if stripped == testDirectoryEntry {
		t.Fatal("executability stripping did not make entry copy")
	} else if stripped.Equal(testDirectoryEntry) {
		t.Fatal("stripped directory entry considered equal to original")
	}

	// Propagate executability from a nil ancestor.
	fixed := PropagateExecutability(nil, stripped)
	if fixed == stripped {
		t.Fatal("executability propagation did not make entry copy")
	} else if !fixed.Equal(stripped) {
		t.Fatal("executability propagation from nil ancestor made changes to entry")
	}

	// Propagate executability from the real ancestor.
	fixed = PropagateExecutability(testDirectoryEntry, stripped)
	if fixed == stripped {
		t.Fatal("executability propagation did not make entry copy")
	} else if !fixed.Equal(testDirectoryEntry) {
		t.Fatal("executability propagation incorrect")
	}
}
