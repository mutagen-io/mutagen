package sync

import (
	"testing"
)

func TestCreationIdentity(t *testing.T) {
	if changes := Diff(nil, gorootSnapshot); len(changes) != 1 {
		t.Fatal("unexpected number of changes")
	} else if changes[0].Path != "" {
		t.Error("unexpected change path")
	} else if changes[0].Old != nil {
		t.Error("unexpected old entry")
	} else if changes[0].New != gorootSnapshot {
		t.Error("unexpected new entry")
	}
}

func TestDeletionIdentity(t *testing.T) {
	if changes := Diff(gorootSnapshot, nil); len(changes) != 1 {
		t.Fatal("unexpected number of changes")
	} else if changes[0].Path != "" {
		t.Error("unexpected change path")
	} else if changes[0].Old != gorootSnapshot {
		t.Error("unexpected old entry")
	} else if changes[0].New != nil {
		t.Error("unexpected new entry")
	}
}
