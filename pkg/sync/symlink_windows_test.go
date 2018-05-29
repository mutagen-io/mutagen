package sync

import (
	"testing"
)

func TestSymlinkWindowsBackslashConversionValid(t *testing.T) {
	if target, err := normalizeSymlinkAndEnsureSane("file", "subdirectory\\other"); err != nil {
		t.Fatal("sane symlink treated as invalid:", err)
	} else if target != "subdirectory/other" {
		t.Error("sane symlink target incorrect:", target, "!=", "subdirectory/other")
	}
}
