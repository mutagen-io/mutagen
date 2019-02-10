package sync

import (
	"testing"
)

func TestTransitionDependenciesEmtpy(t *testing.T) {
	if paths, digests, err := TransitionDependencies(nil); err != nil {
		t.Error("transition dependency finding failed for no transitions:", err)
	} else if len(paths) != 0 {
		t.Error("unexpected number of paths for no transitions")
	} else if len(digests) != len(paths) {
		t.Error("digest count does not match path count")
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
	if _, _, err := TransitionDependencies(transitions); err == nil {
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
	if paths, digests, err := TransitionDependencies(transitions); err != nil {
		t.Error("transition dependency finding failed:", err)
	} else if len(paths) != 0 {
		t.Error("unexpected number of paths")
	} else if len(digests) != len(paths) {
		t.Error("digest count does not match path count")
	}
}

func TestTransitionDependenciesNewNonNil(t *testing.T) {
	transitions := []*Change{
		{
			Path: "",
			New:  testDirectory1Entry,
		},
	}
	if paths, digests, err := TransitionDependencies(transitions); err != nil {
		t.Error("transition dependency finding failed:", err)
	} else if len(paths) != 4 {
		t.Error("unexpected number of paths")
	} else if len(digests) != len(paths) {
		t.Error("digest count does not match path count")
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
	if paths, digests, err := TransitionDependencies(transitions); err != nil {
		t.Error("transition dependency finding failed:", err)
	} else if len(paths) != 0 {
		t.Error("unexpected number of entries")
	} else if len(digests) != len(paths) {
		t.Error("digest count does not match path count")
	}
}
