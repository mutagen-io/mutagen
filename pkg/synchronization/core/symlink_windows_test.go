package core

import (
	"testing"
)

func TestSymlinkWindowsBackslashConversionValid(t *testing.T) {
	if target, err := normalizeSymlinkAndEnsurePortable("file", "subdirectory\\other"); err != nil {
		t.Fatal("portable symlink treated as invalid:", err)
	} else if target != "subdirectory/other" {
		t.Error("portable symlink target incorrect:", target, "!=", "subdirectory/other")
	}
}
