// +build !plan9

// TODO: Figure out what to do for Plan 9. It doesn't have syscall.WaitStatus.

package process

import (
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

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
