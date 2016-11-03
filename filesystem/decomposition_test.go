package filesystem

import (
	"runtime"
	"testing"
)

// TestDecomposesUnicode checks Unicode decomposition behavior against expected
// operating system behaviors.
func TestDecomposesUnicode(t *testing.T) {
	// Compute whether or not the filesystem decomposes unicode. We use an empty
	// string to use the default temporary directory.
	decomposes, err := DecomposesUnicode("")
	if err != nil {
		t.Fatalf("unable to check Unicode decomposition: %s", err)
	}

	// Check whether or not the result jives with what we expect for this
	// operating system. This isn't going to be perfect, it's mostly just a flag
	// for maintainers to validate assumptions about the default behavior of
	// different operating systems and filesystems.
	if runtime.GOOS == "darwin" {
		if !decomposes {
			t.Error("unexpected Unicode normalization preservation")
		}
	} else {
		if decomposes {
			t.Error("unexpected Unicode decomposition")
		}
	}
}
