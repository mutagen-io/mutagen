package sync

import (
	"testing"
)

func TestDiffInternalCreationIdentity(t *testing.T) {
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

func TestDiffInternalDeletionIdentity(t *testing.T) {
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

func TestDiffInternalFileToDirectory(t *testing.T) {
	if changes := diff("", testFile1Entry, testDirectory1Entry); len(changes) != 1 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 1)
	} else if changes[0].Path != "" {
		t.Error("unexpected change path:", changes[0].Path, "!=", "")
	} else if changes[0].Old != testFile1Entry {
		t.Error("unexpected old entry")
	} else if changes[0].New != testDirectory1Entry {
		t.Error("unexpected new entry")
	}
}

func TestDiffInternalDirectoryToFile(t *testing.T) {
	if changes := diff("", testDirectory1Entry, testFile1Entry); len(changes) != 1 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 1)
	} else if changes[0].Path != "" {
		t.Error("unexpected change path:", changes[0].Path, "!=", "")
	} else if changes[0].Old != testDirectory1Entry {
		t.Error("unexpected old entry")
	} else if changes[0].New != testFile1Entry {
		t.Error("unexpected new entry")
	}
}

func TestDiffInternalDirectories(t *testing.T) {
	if changes := diff("", testDirectory1Entry, testDirectory2Entry); len(changes) != 8 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 8)
	}
}

func TestDiffDirectories(t *testing.T) {
	if changes := Diff(testDirectory1Entry, testDirectory2Entry); len(changes) != 8 {
		t.Fatal("unexpected number of changes:", len(changes), "!=", 8)
	}
}
