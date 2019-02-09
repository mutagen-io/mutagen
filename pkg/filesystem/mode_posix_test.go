// +build !windows

package filesystem

import (
	"testing"

	"golang.org/x/sys/unix"
)

// TestPermissionModesMatchExpected verifies that the cross-platform permission
// modes we define match their expected value on each platform. This should be
// guaranteed by a combination of POSIX specifications (on POSIX systems) and
// the Go os package implementation (on Windows).
func TestModePermissionsMaskMatchesOS(t *testing.T) {
	// Verify ModePermissionsMask.
	if ModePermissionsMask != Mode(unix.S_IRWXU|unix.S_IRWXG|unix.S_IRWXO) {
		t.Error("ModePermissionsMask does not match expected value")
	}

	// Verify ModePermissionUserRead.
	if ModePermissionUserRead != Mode(unix.S_IRUSR) {
		t.Error("ModePermissionUserRead does not match expected")
	}

	// Verify ModePermissionUserWrite.
	if ModePermissionUserWrite != Mode(unix.S_IWUSR) {
		t.Error("ModePermissionUserWrite does not match expected")
	}

	// Verify ModePermissionUserExecute.
	if ModePermissionUserExecute != Mode(unix.S_IXUSR) {
		t.Error("ModePermissionUserExecute does not match expected")
	}

	// Verify ModePermissionGroupRead.
	if ModePermissionGroupRead != Mode(unix.S_IRGRP) {
		t.Error("ModePermissionGroupRead does not match expected")
	}

	// Verify ModePermissionGroupWrite.
	if ModePermissionGroupWrite != Mode(unix.S_IWGRP) {
		t.Error("ModePermissionGroupWrite does not match expected")
	}

	// Verify ModePermissionGroupExecute.
	if ModePermissionGroupExecute != Mode(unix.S_IXGRP) {
		t.Error("ModePermissionGroupExecute does not match expected")
	}

	// Verify ModePermissionOthersRead.
	if ModePermissionOthersRead != Mode(unix.S_IROTH) {
		t.Error("ModePermissionOthersRead does not match expected")
	}

	// Verify ModePermissionOthersWrite.
	if ModePermissionOthersWrite != Mode(unix.S_IWOTH) {
		t.Error("ModePermissionOthersWrite does not match expected")
	}

	// Verify ModePermissionOthersExecute.
	if ModePermissionOthersExecute != Mode(unix.S_IXOTH) {
		t.Error("ModePermissionOthersExecute does not match expected")
	}
}
