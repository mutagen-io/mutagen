package core

import (
	"testing"
)

// stripExecutabilityRecursive is used by stripExecutability for recursion.
func stripExecutabilityRecursive(snapshot *Entry) {
	// If the entry is nil, then there's nothing to strip.
	if snapshot == nil {
		return
	}

	// Handle the stripping based on entry kind.
	if snapshot.Kind == EntryKind_Directory {
		for _, entry := range snapshot.Contents {
			stripExecutabilityRecursive(entry)
		}
	} else if snapshot.Kind == EntryKind_File {
		snapshot.Executable = false
	}
}

// stripExecutability creates a deep copy of an entry with executability
// information removed.
func stripExecutability(snapshot *Entry) *Entry {
	// Create a copy of the snapshot that we can mutate.
	result := snapshot.Copy(EntryCopyBehaviorDeep)

	// Perform stripping.
	stripExecutabilityRecursive(result)

	// Done.
	return result
}

// TestPropagateExecutability tests PropagateExecutability.
func TestPropagateExecutability(t *testing.T) {
	// Test propagation to a nil target.
	if PropagateExecutability(tDMU, tDMU, nil) != nil {
		t.Error("executability propagation to nil entry did not return nil")
	}

	// Create a copy of the test directory entry with executability stripped and
	// ensure that it differs.
	stripped := stripExecutability(tDMU)
	if stripped == tDMU {
		t.Fatal("executability stripping did not make entry copy")
	} else if stripped.Equal(tDMU, true) {
		t.Error("stripped directory entry considered equal to original")
	}

	// Propagate executability from a nil ancestor/source. This should have no
	// effect on the executability-stripped contents.
	fixed := PropagateExecutability(nil, nil, stripped)
	if fixed == stripped {
		t.Fatal("executability propagation did not make entry copy")
	} else if !fixed.Equal(stripped, true) {
		t.Error("executability propagation from nil ancestor/source made changes to entry")
	}

	// Propagate from an ancestor and ensure that executability is restored.
	stripped = stripExecutability(tDMU)
	fixed = PropagateExecutability(tDMU, nil, stripped)
	if fixed == stripped {
		t.Fatal("executability propagation did not make entry copy")
	} else if !fixed.Equal(tDMU, true) {
		t.Error("executability propagation from ancestor incorrect")
	}

	// Propagate from a preserving source and ensure that executability is
	// restored.
	stripped = stripExecutability(tDMU)
	fixed = PropagateExecutability(nil, tDMU, stripped)
	if fixed == stripped {
		t.Fatal("executability propagation did not make entry copy")
	} else if !fixed.Equal(tDMU, true) {
		t.Error("executability propagation from source incorrect")
	}

	// Propagate from a preserving source and ancestor in the case where the
	// non-preserving side has modified the file.
	//
	// HACK: We know that stripExecutability generates a deep copy of its input
	// value, so we can modify it here.
	stripped = stripExecutability(tDMU)
	stripped.Contents["executable file"] = tF2
	fixed = PropagateExecutability(tDMU, tDMU, stripped)
	expected := tF2.Copy(EntryCopyBehaviorShallow)
	expected.Executable = true
	if !fixed.Contents["executable file"].Equal(expected, true) {
		t.Error("executability propagation from ancestor and source incorrect")
	}
}
