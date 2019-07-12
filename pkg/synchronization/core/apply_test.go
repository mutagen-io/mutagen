package core

import (
	"testing"
)

func TestApplyRootSwap(t *testing.T) {
	// Create a change the swaps out the root entry.
	changes := []*Change{
		{
			Old: testDirectory1Entry,
			New: testFile1Entry,
		},
	}

	// Ensure that the swap is applied correctly.
	if result, err := Apply(testDirectory1Entry, changes); err != nil {
		t.Fatal("unable to apply changes:", err)
	} else if !result.Equal(testFile1Entry) {
		t.Error("mismatch after root replacement")
	}
}

func TestApplyDiff(t *testing.T) {
	// Compute the diff between two different directories.
	changes := diff("", testDirectory1Entry, testDirectory2Entry)

	// Test that applying the diff to the base results in the target.
	if result, err := Apply(testDirectory1Entry, changes); err != nil {
		t.Fatal("unable to apply changes:", err)
	} else if !result.Equal(testDirectory2Entry) {
		t.Error("mismatch after diff/apply cycle")
	}
}

func TestApplyMissingParentPath(t *testing.T) {
	// Create a change with an invalid path.
	changes := []*Change{
		{
			Path: "this/does/not/exist",
			New:  testFile1Entry,
		},
	}

	// Ensure that application of the change fails.
	if _, err := Apply(testDirectory1Entry, changes); err == nil {
		t.Fatal("change referencing invalid path did not fail to apply")
	}
}
