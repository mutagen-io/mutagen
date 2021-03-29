package core

import (
	"testing"
)

// TestChangeEnsureValid tests Change.EnsureValid.
func TestChangeEnsureValid(t *testing.T) {
	// Ensure that a nil change is considered invalid in all cases.
	var change *Change
	if change.EnsureValid(false) == nil {
		t.Error("nil change incorrectly classified as valid (without synchronizability requirement)")
	}
	if change.EnsureValid(true) == nil {
		t.Error("nil change incorrectly classified as valid (when requiring synchronizability)")
	}

	// Process test cases.
	for i, test := range entryEnsureValidTestCases {
		// Compute a description for the test in case we need it.
		description := "without synchronizability requirement"
		if test.synchronizable {
			description = "when requiring synchronizability"
		}

		// Check validity in the case of deletion.
		deletion := &Change{Old: test.entry}
		err := deletion.EnsureValid(test.synchronizable)
		valid := err == nil
		if valid != test.expected {
			if valid {
				t.Errorf("test index %d: deletion change incorrectly classified as valid (%s)", i, description)
			} else {
				t.Errorf("test index %d: deletion change incorrectly classified as invalid (%s): %v", i, description, err)
			}
		}

		// Check validity in the case of creation.
		creation := &Change{New: test.entry}
		err = creation.EnsureValid(test.synchronizable)
		valid = err == nil
		if valid != test.expected {
			if valid {
				t.Errorf("test index %d: creation change incorrectly classified as valid (%s)", i, description)
			} else {
				t.Errorf("test index %d: creation change incorrectly classified as invalid (%s): %v", i, description, err)
			}
		}
	}
}

// TestChangeSlim tests Change.slim.
func TestChangeSlim(t *testing.T) {
	// Define test cases.
	tests := []struct {
		change   *Change
		expected *Change
	}{
		{&Change{}, &Change{}},
		{&Change{Path: "content", Old: tF1}, &Change{Path: "content", Old: tF1}},
		{&Change{Path: "content", New: tF1}, &Change{Path: "content", New: tF1}},
		{&Change{Path: "content", Old: tD1}, &Change{Path: "content", Old: tD0}},
		{&Change{Path: "content", New: tD1}, &Change{Path: "content", New: tD0}},
		{&Change{Old: tSA, New: tD1}, &Change{Old: tSA, New: tD0}},
		{&Change{Old: tD1, New: tSA}, &Change{Old: tD0, New: tSA}},
	}

	// Process test cases.
	for i, test := range tests {
		// Compute the slim version of the change and ensure that it's non-nil.
		slim := test.change.slim()
		if slim == nil {
			t.Errorf("test index %d: slimmed change is nil", i)
			continue
		}

		// Verify that the path matches what's expected.
		if slim.Path != test.expected.Path {
			t.Errorf("test index %d: old value does not match expected", i)
		}

		// Verify that the old value matches what's expected.
		if !slim.Old.Equal(test.expected.Old, true) {
			t.Errorf("test index %d: old value does not match expected", i)
		}

		// Verify that the new value matches what's expected.
		if !slim.New.Equal(test.expected.New, true) {
			t.Errorf("test index %d: new value does not match expected", i)
		}
	}
}

// TestIsRootDeletion tests Change.IsRootDeletion.
func TestIsRootDeletion(t *testing.T) {
	// Define test cases.
	tests := []struct {
		change   *Change
		expected bool
	}{
		{&Change{New: tD1}, false},
		{&Change{Old: tF1, New: tD1}, false},
		{&Change{}, false},
		{&Change{Old: tF1, New: tF1}, false},
		{&Change{Old: tF1}, true},
		{&Change{Old: tD1}, true},
	}

	// Process test cases.
	for i, test := range tests {
		isRootDeletion := test.change.IsRootDeletion()
		if isRootDeletion && !test.expected {
			t.Errorf("test index %d: incorrectly classified as root deletion", i)
		} else if !isRootDeletion && test.expected {
			t.Errorf("test index %d: not correctly classified as root deletion", i)
		}
	}
}

// TestIsRootTypeChange tests Change.IsRootTypeChange.
func TestIsRootTypeChange(t *testing.T) {
	// Define test cases.
	tests := []struct {
		change   *Change
		expected bool
	}{
		{&Change{}, false},
		{&Change{Old: tF1}, false},
		{&Change{Old: tD1}, false},
		{&Change{Old: tF1, New: tF1}, false},
		{&Change{Old: tD1, New: tD1}, false},
		{&Change{Old: tF1, New: tF2}, false},
		{&Change{Old: tD1, New: tD2}, false},
		{&Change{Old: tF1, New: tD1}, true},
		{&Change{Old: tD1, New: tF1}, true},
	}

	// Process test cases.
	for i, test := range tests {
		isRootTypeChange := test.change.IsRootTypeChange()
		if isRootTypeChange && !test.expected {
			t.Errorf("test index %d: incorrectly classified as root type change", i)
		} else if !isRootTypeChange && test.expected {
			t.Errorf("test index %d: not correctly classified as root type change", i)
		}
	}
}
