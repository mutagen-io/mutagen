package filesystem

import (
	"os"
	"testing"
)

const (
	// testingDirectoryName is the name of a testing directory to create within
	// the Mutagen data directory.
	testingDirectoryName = "testing"
)

// TestMutagen tests the Mutagen data directory creation function.
func TestMutagen(t *testing.T) {
	// Attempt to create the testing subdirectory and defer its removal.
	path, err := Mutagen(true, testingDirectoryName)
	if err != nil {
		t.Fatal("unable to create testing subdirectory:", err)
	}
	defer os.RemoveAll(path)

	// Ensure it exists and is a directory.
	if info, err := os.Lstat(path); err != nil {
		t.Fatal("unable to probe testing subdirectory:", err)
	} else if !info.IsDir() {
		t.Error("Mutagen subpath is not a directory")
	}
}
