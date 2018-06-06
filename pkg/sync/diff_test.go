package sync

import (
	"testing"
)

func TestDiffCreationIdentity(t *testing.T) {
	if changes := diff("", nil, testDirectory1Entry); len(changes) != 1 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 1)
	} else if changes[0].Path != "" {
		t.Error("unexpected change path:", changes[0].Path, "!=", "")
	} else if changes[0].Old != nil {
		t.Error("unexpected old entry")
	} else if changes[0].New != testDirectory1Entry {
		t.Error("unexpected new entry")
	}
}

func TestDiffDeletionIdentity(t *testing.T) {
	if changes := diff("", testDirectory1Entry, nil); len(changes) != 1 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 1)
	} else if changes[0].Path != "" {
		t.Error("unexpected change path:", changes[0].Path, "!=", "")
	} else if changes[0].Old != testDirectory1Entry {
		t.Error("unexpected old entry")
	} else if changes[0].New != nil {
		t.Error("unexpected new entry")
	}
}

func TestDiffFileToDirectory(t *testing.T) {
	if changes := diff("", testFileEntry, testDirectory1Entry); len(changes) != 1 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 1)
	} else if changes[0].Path != "" {
		t.Error("unexpected change path:", changes[0].Path, "!=", "")
	} else if changes[0].Old != testFileEntry {
		t.Error("unexpected old entry")
	} else if changes[0].New != testDirectory1Entry {
		t.Error("unexpected new entry")
	}
}

func TestDiffDirectoryToFile(t *testing.T) {
	if changes := diff("", testDirectory1Entry, testFileEntry); len(changes) != 1 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 1)
	} else if changes[0].Path != "" {
		t.Error("unexpected change path:", changes[0].Path, "!=", "")
	} else if changes[0].Old != testDirectory1Entry {
		t.Error("unexpected old entry")
	} else if changes[0].New != testFileEntry {
		t.Error("unexpected new entry")
	}
}

func TestDiffDirectories(t *testing.T) {
	if changes := diff("", testDirectory1Entry, testDirectory2Entry); len(changes) != 8 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 8)
	}
}
