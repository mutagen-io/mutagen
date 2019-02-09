package filesystem

import (
	"os"
	"testing"
)

// TestPermissionModesMatchExpected verifies that the cross-platform permission
// modes we define match their expected value on each platform. This should be
// guaranteed by a combination of POSIX specifications (on POSIX systems) and
// the Go os package implementation (on Windows).
func TestModePermissionsMaskMatchesOS(t *testing.T) {
	if ModePermissionsMask != Mode(os.ModePerm) {
		t.Error("ModePermissionsMask does not match expected value")
	}
}
