package sync

import (
	"testing"
)

func TestTransitionDependenciesEmtpy(t *testing.T) {
	if entries, err := TransitionDependencies(nil); err != nil {
		t.Error("transition dependency finding failed for no transitions:", err)
	} else if len(entries) != 0 {
		t.Error("unexpected number of entries for no transitions")
	}
}

func TestTransitionDependenciesInvalid(t *testing.T) {
	root := testDirectory1Entry.Copy()
	root.Contents["directory"].Contents["subfile"].Kind = (EntryKind_Symlink + 1)
	transitions := []*Change{
		{
			Path: "",
			New:  root,
		},
	}
	if _, err := TransitionDependencies(transitions); err == nil {
		t.Error("transition dependency finding succeeded for invalid transition")
	}
}

func TestTransitionDependenciesNewNil(t *testing.T) {
	transitions := []*Change{
		{
			Path: "",
			New:  nil,
		},
	}
	if entries, err := TransitionDependencies(transitions); err != nil {
		t.Error("transition dependency finding failed:", err)
	} else if len(entries) != 0 {
		t.Error("unexpected number of entries")
	}
}

func TestTransitionDependenciesNewNonNil(t *testing.T) {
	transitions := []*Change{
		{
			Path: "",
			New:  testDirectory1Entry,
		},
	}
	if entries, err := TransitionDependencies(transitions); err != nil {
		t.Error("transition dependency finding failed:", err)
	} else if len(entries) != 4 {
		t.Error("unexpected number of entries")
	}
}

func TestTransitionDependenciesOnlyExecutableBitChange(t *testing.T) {
	old := testFile2Entry.Copy()
	old.Executable = false
	transitions := []*Change{
		{
			Path: "",
			Old:  old,
			New:  testFile2Entry,
		},
	}
	if entries, err := TransitionDependencies(transitions); err != nil {
		t.Error("transition dependency finding failed:", err)
	} else if len(entries) != 0 {
		t.Error("unexpected number of entries")
	}
}
