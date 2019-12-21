package core

import (
	"testing"
)

func TestChangeCopySlim(t *testing.T) {
	// Create a sample change.
	change := &Change{
		Path: "test",
		Old:  nil,
		New:  testDirectory2Entry,
	}

	// Create a slim copy.
	slim := change.copySlim()

	// Check validity.
	if err := slim.EnsureValid(); err != nil {
		t.Fatal("slim copy of change is invalid:", err)
	}

	// Check path.
	if slim.Path != "test" {
		t.Error("slim copy of change has differing path")
	}

	// Check old entry.
	if !slim.Old.Equal(nil) {
		t.Error("slim copy of change has incorrect old entry")
	}

	// Check new entry.
	if !slim.New.Equal(testEmptyDirectory) {
		t.Error("slim copy of change has incorrect new entry")
	}
}

func TestChangeNilInvalid(t *testing.T) {
	var change *Change
	if change.EnsureValid() == nil {
		t.Error("nil change considered valid")
	}
}

func TestChangeValid(t *testing.T) {
	change := &Change{New: testSymlinkEntry}
	if err := change.EnsureValid(); err != nil {
		t.Error("valid change considered invalid:", err)
	}
}

// TestIsRootDeletion tests that Change.IsRootDeletion behaves as expected.
func TestIsRootDeletion(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		change               *Change
		expectIsRootDeletion bool
	}{
		{&Change{New: testDirectory1Entry}, false},
		{&Change{Old: testFile1Entry, New: testDirectory1Entry}, false},
		{&Change{}, false},
		{&Change{Old: testFile1Entry, New: testFile1Entry}, false},
		{&Change{Old: testFile1Entry}, true},
		{&Change{Old: testDirectory1Entry}, true},
	}

	// Process test cases.
	for _, testCase := range testCases {
		isRootDeletion := testCase.change.IsRootDeletion()
		if isRootDeletion && !testCase.expectIsRootDeletion {
			t.Error("test case incorrectly classified as root deletion")
		} else if !isRootDeletion && testCase.expectIsRootDeletion {
			t.Error("test case not correctly classified as root deletion")
		}
	}
}

// TestIsRootTypeChange tests that Change.IsRootTypeChange behaves as expected.
func TestIsRootTypeChange(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		change                 *Change
		expectIsRootTypeChange bool
	}{
		{&Change{}, false},
		{&Change{Old: testFile1Entry}, false},
		{&Change{Old: testDirectory1Entry}, false},
		{&Change{Old: testFile1Entry, New: testFile1Entry}, false},
		{&Change{Old: testDirectory1Entry, New: testDirectory1Entry}, false},
		{&Change{Old: testFile1Entry, New: testFile2Entry}, false},
		{&Change{Old: testDirectory1Entry, New: testDirectory2Entry}, false},
		{&Change{Old: testFile1Entry, New: testDirectory1Entry}, true},
		{&Change{Old: testDirectory1Entry, New: testFile1Entry}, true},
	}

	// Process test cases.
	for _, testCase := range testCases {
		isRootTypeChange := testCase.change.IsRootTypeChange()
		if isRootTypeChange && !testCase.expectIsRootTypeChange {
			t.Error("test case incorrectly classified as root type change")
		} else if !isRootTypeChange && testCase.expectIsRootTypeChange {
			t.Error("test case not correctly classified as root type change")
		}
	}
}
