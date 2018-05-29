package sync

import (
	"testing"
)

func TestDiffCreationIdentity(t *testing.T) {
	if changes := diff("", nil, testDirectoryEntry); len(changes) != 1 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 1)
	} else if changes[0].Path != "" {
		t.Error("unexpected change path:", changes[0].Path, "!=", "")
	} else if changes[0].Old != nil {
		t.Error("unexpected old entry")
	} else if changes[0].New != testDirectoryEntry {
		t.Error("unexpected new entry")
	}
}

func TestDiffDeletionIdentity(t *testing.T) {
	if changes := diff("", testDirectoryEntry, nil); len(changes) != 1 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 1)
	} else if changes[0].Path != "" {
		t.Error("unexpected change path:", changes[0].Path, "!=", "")
	} else if changes[0].Old != testDirectoryEntry {
		t.Error("unexpected old entry")
	} else if changes[0].New != nil {
		t.Error("unexpected new entry")
	}
}

func TestDiffFileToDirectory(t *testing.T) {
	if changes := diff("", testFileEntry, testDirectoryEntry); len(changes) != 1 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 1)
	} else if changes[0].Path != "" {
		t.Error("unexpected change path:", changes[0].Path, "!=", "")
	} else if changes[0].Old != testFileEntry {
		t.Error("unexpected old entry")
	} else if changes[0].New != testDirectoryEntry {
		t.Error("unexpected new entry")
	}
}

func TestDiffDirectoryToFile(t *testing.T) {
	if changes := diff("", testDirectoryEntry, testFileEntry); len(changes) != 1 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 1)
	} else if changes[0].Path != "" {
		t.Error("unexpected change path:", changes[0].Path, "!=", "")
	} else if changes[0].Old != testDirectoryEntry {
		t.Error("unexpected old entry")
	} else if changes[0].New != testFileEntry {
		t.Error("unexpected new entry")
	}
}

func TestDiffDirectories(t *testing.T) {
	if changes := diff("", testDirectoryEntry, testAlternateDirectoryEntry); len(changes) != 8 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 8)
	}
}
