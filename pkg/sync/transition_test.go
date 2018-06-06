package sync

import (
	"os"
	"runtime"
	"testing"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

func testTransitionCycle(entry *Entry, contentMap map[string][]byte) error {
	// Create test content on disk and defer its removal. This will exercise
	// the creation portion of Transition.
	root, parent, err := createTestContentOnDisk(entry, contentMap)
	if err != nil {
		return errors.Wrap(err, "unable to create test content")
	}
	defer os.RemoveAll(parent)

	// Grab the expected entry. If we're on a system that doesn't support
	// executability, then strip executability from the expected value.
	expected := entry
	if !filesystem.PreservesExecutability {
		expected = StripExecutability(expected)
	}

	// Create a hasher.
	hasher := newTestHasher()

	// Perform a scan.
	snapshot, cache, err := Scan(root, hasher, nil, nil, SymlinkMode_Sane)
	if err != nil {
		return errors.Wrap(err, "unable to perform scan")
	} else if cache == nil {
		return errors.New("nil cache returned")
	} else if !snapshot.Equal(expected) {
		return errors.New("snapshot not equal to expected")
	}

	// Set up transitions to remove the expected content.
	transitions := []*Change{{Old: expected}}

	// Perform the removal transition.
	if entries, problems := Transition(root, transitions, cache, SymlinkMode_Sane, nil); len(problems) != 0 {
		return errors.New("problems occurred during removal transition")
	} else if len(entries) != 1 {
		return errors.New("unexpected number of entries returned from removal transition")
	} else if entries[0] != nil {
		return errors.New("removed entry does not match expected")
	}

	// Success.
	return nil
}

func TestTransitionNilRoot(t *testing.T) {
	if err := testTransitionCycle(testNilEntry, nil); err != nil {
		t.Error("transition cycle failed:", err)
	}
}

func TestTransitionFile1Root(t *testing.T) {
	if err := testTransitionCycle(testFile1Entry, testFile1ContentMap); err != nil {
		t.Error("transition cycle failed:", err)
	}
}

func TestTransitionFile2Root(t *testing.T) {
	if err := testTransitionCycle(testFile2Entry, testFile2ContentMap); err != nil {
		t.Error("transition cycle failed:", err)
	}
}

func TestTransitionFile3Root(t *testing.T) {
	if err := testTransitionCycle(testFile3Entry, testFile3ContentMap); err != nil {
		t.Error("transition cycle failed:", err)
	}
}

func TestTransitionDirectory1Root(t *testing.T) {
	if err := testTransitionCycle(testDirectory1Entry, testDirectory1ContentMap); err != nil {
		t.Error("transition cycle failed:", err)
	}
}

func TestTransitionDirectory2Root(t *testing.T) {
	if err := testTransitionCycle(testDirectory2Entry, testDirectory2ContentMap); err != nil {
		t.Error("transition cycle failed:", err)
	}
}

func TestTransitionDirectory3Root(t *testing.T) {
	if err := testTransitionCycle(testDirectory3Entry, testDirectory3ContentMap); err != nil {
		t.Error("transition cycle failed:", err)
	}
}

func TestTransitionCaseConflict(t *testing.T) {
	// HACK: We actually ought to be determining this based on the filesystem
	// being used, but it's a sufficient test mechanism for now.
	expectCaseConflict := runtime.GOOS == "windows" || runtime.GOOS == "darwin"
	err := testTransitionCycle(testDirectoryWithCaseConflict, testDirectoryWithCaseConflictContentMap)
	if expectCaseConflict && err == nil {
		t.Error("expected case conflict")
	} else if !expectCaseConflict && err != nil {
		t.Error("unexpected case conflict")
	}
}
