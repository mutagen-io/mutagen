package process

import (
	"os/exec"
	"runtime"
	"testing"
)

// TestIsPOSIXShellInvalidCommand tests that the IsPOSIXShellInvalidCommand
// function correctly identifiers an "invalid command" error from a POSIX shell.
func TestIsPOSIXShellInvalidCommand(t *testing.T) {
	// If we're not running in a POSIX environment, then skip this test. I think
	// that we also have to skip this test in POSIX environments on Windows
	// (which might be detectable with, e.g., the go-isatty package), because Go
	// won't be able to find shell paths (e.g. "/bin/sh") due to how it resolves
	// executable paths.
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	// Attempt to run a command that doesn't exist and verify that it has the
	// correct error classification. Note that we have to run this inside a
	// shell, otherwise other errors will crop up before the shell's error.
	command := exec.Command("/bin/sh", "-c", "/dev/null")
	if err := command.Run(); err == nil {
		t.Fatal("expected non-nil error when running invalid command")
	} else if !IsPOSIXShellInvalidCommand(command.ProcessState) {
		t.Error("expected POSIX invalid command classification")
	}
}

// TestIsPOSIXShellCommandNotFound tests that the IsPOSIXShellCommandNotFound
// function correctly identifiers a "command not found" error from a POSIX
// shell.
func TestIsPOSIXShellCommandNotFound(t *testing.T) {
	// If we're not running in a POSIX environment, then skip this test. I think
	// that we also have to skip this test in POSIX environments on Windows
	// (which might be detectable with, e.g., the go-isatty package), because Go
	// won't be able to find shell paths (e.g. "/bin/sh") due to how it resolves
	// executable paths.
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	// Attempt to run a command that doesn't exist and verify that it has the
	// correct error classification. Note that we have to run this inside a
	// shell, otherwise other errors will crop up before the shell's error.
	command := exec.Command("/bin/sh", "mutagen-test-not-exist")
	if err := command.Run(); err == nil {
		t.Fatal("expected non-nil error when running non-existent command")
	} else if !IsPOSIXShellCommandNotFound(command.ProcessState) {
		t.Error("expected POSIX command not found classification")
	}
}
