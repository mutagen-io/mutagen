package filesystem

import (
	"runtime"
	"testing"
)

// TestCaseInsensitive checks case insensitivity behavior against expected
// operating system behaviors.
func TestCaseInsensitive(t *testing.T) {
	// Compute whether or not the filesystem is case sensitive. We use an empty
	// string to use the default temporary directory.
	insensitive, err := CaseInsensitive("")
	if err != nil {
		t.Fatalf("unable to check case sensitivity: %s", err)
	}

	// Check whether or not the result jives with what we expect for this
	// operating system. This isn't going to be perfect, it's mostly just a flag
	// for maintainers to validate assumptions about the default behavior of
	// different operating systems and filesystems.
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		if !insensitive {
			t.Error("unexpected case sensitivity")
		}
	} else {
		if insensitive {
			t.Error("unexpected case insensitivity")
		}
	}
}
