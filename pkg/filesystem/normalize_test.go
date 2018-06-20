package filesystem

import (
	"path/filepath"
	"testing"
)

func TestNormalizeHome(t *testing.T) {
	// Compute a path relative to the home directory.
	normalized, err := Normalize("~/somepath")
	if err != nil {
		t.Fatal("unable to normalize path:", err)
	}

	// Ensure that it's what we expect.
	if normalized != filepath.Join(HomeDirectory, "somepath") {
		t.Error("normalized path does not match expected")
	}
}
