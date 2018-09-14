// +build !plan9

// TODO: Figure out what to do for Plan 9. It doesn't have syscall.WaitStatus.

package process

import (
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

const (
	// posixShellInvalidCommandExitCode is the exit code returned by most (all?)
	// POSIX shells when the provided command is invalid, e.g. due to an file
	// without executable permissions. It seems to have originated with the
	// Bourne shell and then been brought over to bash, zsh, and others. It
	// doesn't seem to have a corresponding errno value, which I guess makes
	// sense since errno values aren't generally expected to be used as exit
	// codes, so we have to define it manually.
	// TODO: Figure out if other shells return different exit codes when a
	// command isn't found. Is this exit code defined in a standard somewhere?
	posixShellInvalidCommandExitCode = 126

	// posixShellCommandNotFoundExitCode is the exit code returned by most
	// (all?) POSIX shells when the provided command isn't found. It seems to
	// have originated with the Bourne shell and then been brought over to bash,
	// zsh, and others. It doesn't seem to have a corresponding errno value,
	// which I guess makes sense since errno values aren't generally expected to
	// be used as exit codes, so we have to define it manually.
	// TODO: Figure out if other shells return different exit codes when a
	// command isn't found. Is this exit code defined in a standard somewhere?
	posixShellCommandNotFoundExitCode = 127
)

// ExitCodeForError extracts the process exit code from an error returned by
// os/exec.Process.Wait/Run. The error must be of type *os/exec.ExitError in
// order for this function to succeed.
func ExitCodeForError(err error) (int, error) {
	// Attempt to extract the error.
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return 0, errors.New("error is not an exit error")
	}

	// Attempt to extract the wait status. The syscall.WaitStatus type is
	// platform-dependent, but this code uses a portable subset of its features.
	waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		return 0, errors.New("unable to access wait status")
	}

	// Done.
	return waitStatus.ExitStatus(), nil
}

// IsPOSIXShellInvalidCommand returns whether or not an os/exec error represents
// an "invalid" error from a POSIX shell.
func IsPOSIXShellInvalidCommand(err error) bool {
	// Extract the code.
	code, codeErr := ExitCodeForError(err)

	// Ensure that extraction was successful and the code matches what's
	// expected.
	return codeErr == nil && code == posixShellInvalidCommandExitCode
}

// IsPOSIXShellCommandNotFound returns whether or not an os/exec error
// represents a "command not found" error from a POSIX shell.
func IsPOSIXShellCommandNotFound(err error) bool {
	// Extract the code.
	code, codeErr := ExitCodeForError(err)

	// Ensure that extraction was successful and the code matches what's
	// expected.
	return codeErr == nil && code == posixShellCommandNotFoundExitCode
}
