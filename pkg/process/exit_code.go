// +build !plan9

// TODO: Figure out what to do for Plan 9. It doesn't have syscall.WaitStatus.

package process

import (
	"os"
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

// ExitCodeForProcessState extracts the process exit code from the process'
// post-exit state.
func ExitCodeForProcessState(state *os.ProcessState) (int, error) {
	// Attempt to extract the wait status. The syscall.WaitStatus type is
	// platform-dependent, but this code uses a portable subset of its features.
	waitStatus, ok := state.Sys().(syscall.WaitStatus)
	if !ok {
		return 0, errors.New("unable to access wait status")
	}

	// Done.
	return waitStatus.ExitStatus(), nil
}

// IsPOSIXShellInvalidCommand returns whether or not a process state represents
// an "invalid" error from a POSIX shell.
func IsPOSIXShellInvalidCommand(state *os.ProcessState) bool {
	// Extract the code.
	code, err := ExitCodeForProcessState(state)

	// Ensure that extraction was successful and the code matches what's
	// expected.
	return err == nil && code == posixShellInvalidCommandExitCode
}

// IsPOSIXShellCommandNotFound returns whether or not a process state represents
// a "command not found" error from a POSIX shell.
func IsPOSIXShellCommandNotFound(state *os.ProcessState) bool {
	// Extract the code.
	code, err := ExitCodeForProcessState(state)

	// Ensure that extraction was successful and the code matches what's
	// expected.
	return err == nil && code == posixShellCommandNotFoundExitCode
}
