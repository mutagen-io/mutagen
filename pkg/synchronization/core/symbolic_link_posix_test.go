//go:build !windows

package core

import (
	"testing"
)

func TestSymbolicLinkPOSIXBackslashInvalid(t *testing.T) {
	if _, err := normalizeSymbolicLinkAndEnsurePortable("file", "target\\path"); err == nil {
		t.Fatal("symbolic link with backslash in target treated as sane")
	}
}
