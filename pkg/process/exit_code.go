package process

import (
	"os"
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

// IsPOSIXShellInvalidCommand returns whether or not a process state represents
// an "invalid" error from a POSIX shell.
func IsPOSIXShellInvalidCommand(state *os.ProcessState) bool {
	return state.ExitCode() == posixShellInvalidCommandExitCode
}

// IsPOSIXShellCommandNotFound returns whether or not a process state represents
// a "command not found" error from a POSIX shell.
func IsPOSIXShellCommandNotFound(state *os.ProcessState) bool {
	return state.ExitCode() == posixShellCommandNotFoundExitCode
}
