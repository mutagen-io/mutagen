package core

import (
	"testing"
)

func TestSymbolicLinkWindowsBackslashConversionValid(t *testing.T) {
	if target, err := normalizeSymbolicLinkAndEnsurePortable("file", "subdirectory\\other"); err != nil {
		t.Fatal("portable symbolic link treated as invalid:", err)
	} else if target != "subdirectory/other" {
		t.Error("portable symbolic link target incorrect:", target, "!=", "subdirectory/other")
	}
}
