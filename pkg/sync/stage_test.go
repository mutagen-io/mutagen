package sync

import (
	"testing"
)

func TestTransitionDependenciesEmtpy(t *testing.T) {
	if names, entries, err := TransitionDependencies(nil); err != nil {
		t.Error("transition dependency finding failed for no transitions:", err)
	} else if len(names) != 0 {
		t.Error("unexpected number of names for no transitions")
	} else if len(entries) != 0 {
		t.Error("unexpected number of entries for no transitions")
	}
}

func TestTransitionDependenciesInvalid(t *testing.T) {
	transitions := []*Change{
		{
			Path: "",
			New: &Entry{
				Kind: (EntryKind_Symlink + 1),
			},
		},
	}
	if _, _, err := TransitionDependencies(transitions); err == nil {
		t.Error("transition dependency finding succeeded for invalid transition")
	}
}

func TestTransitionDependencies(t *testing.T) {
	transitions := []*Change{
		{
			Path: "",
			New:  testDirectory1Entry,
		},
	}
	if names, entries, err := TransitionDependencies(transitions); err != nil {
		t.Error("transition dependency finding failed:", err)
	} else if len(names) != 4 {
		t.Error("unexpected number of names")
	} else if len(entries) != 4 {
		t.Error("unexpected number of entries")
	}
}
