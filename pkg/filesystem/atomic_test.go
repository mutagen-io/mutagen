package filesystem

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicNonExistentDirectory(t *testing.T) {
	if WriteFileAtomic("/does/not/exist", []byte{}, 0600) == nil {
		t.Error("atomic file write did not fail for non-existent path")
	}
}

func TestWriteFileAtomic(t *testing.T) {
	// Compute the target path.
	target := filepath.Join(t.TempDir(), "file")

	// Create contents.
	contents := []byte{0, 1, 2, 3, 4, 5, 6}

	// Attempt to write to a temporary file.
	if err := WriteFileAtomic(target, contents, 0600); err != nil {
		t.Fatal("atomic file write failed:", err)
	}

	// Read the contents back and ensure they match what's expected.
	if data, err := os.ReadFile(target); err != nil {
		t.Fatal("unable to read back file:", err)
	} else if !bytes.Equal(data, contents) {
		t.Error("file contents did not match expected")
	}
}
