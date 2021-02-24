// +build !windows,!plan9

// TODO: Figure out what to do for Plan 9. It doesn't support FcntlFlock at all,
// but we might be able to ~emulate it with os.O_EXCL, but that wouldn't allow
// us to automatically release locks if a process dies.

package locking

import (
	"os"

	"golang.org/x/sys/unix"

	"github.com/pkg/errors"
)

// fcntlFlockRetryingOnEINTR is a wrapper around the fcntl system call that
// retries on EINTR errors and returns on the first successful call or non-EINTR
// error.
func fcntlFlockRetryingOnEINTR(file uintptr, command int, specification *unix.Flock_t) error {
	for {
		err := unix.FcntlFlock(file, command, specification)
		if err == unix.EINTR {
			continue
		}
		return err
	}
}

// Lock attempts to acquire the file lock.
func (l *Locker) Lock(block bool) error {
	// Verify that we don't already hold the lock.
	if l.held {
		return errors.New("lock already held")
	}

	// Set up the lock specification.
	lockSpec := unix.Flock_t{
		Type:   unix.F_WRLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}

	// Set up the blocking specification.
	operation := unix.F_SETLK
	if block {
		operation = unix.F_SETLKW
	}

	// Attempt to perform locking, retrying if EINTR is encountered. According
	// to the POSIX standard, EINTR should only be expected in blocking cases
	// (i.e. when using F_SETLKW), but Linux allows it to be received when using
	// F_SETLK if it occurs before the lock is checked or acquired. Given that
	// Go's runtime preemption can also cause spurious interrupts, it's best to
	// handle EINTR in all cases.
	if err := fcntlFlockRetryingOnEINTR(l.file.Fd(), operation, &lockSpec); err != nil {
		return err
	}

	// Mark the lock as held.
	l.held = true

	// Success.
	return nil
}

// Unlock releases the file lock.
func (l *Locker) Unlock() error {
	// Verify that we hold the lock.
	if !l.held {
		return errors.New("lock not held")
	}

	// Set up the unlock specification.
	unlockSpec := unix.Flock_t{
		Type:   unix.F_UNLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}

	// Attempt to perform unlocking, retrying if EINTR is encountered. According
	// to the POSIX standard, EINTR should only be expected in blocking cases
	// (i.e. when using F_SETLKW), but Linux allows it to be received in certain
	// cases when using F_SETLK. Given that Go's runtime preemption can also
	// cause spurious interrupts, it's best to handle EINTR in all cases.
	//
	// One thing worth considering here is the fact that neither POSIX nor Linux
	// makes any guarantees about the state of the lock if EINTR is received. In
	// theory, the lock could be unlocked, in which case continuing to retry
	// unlocking could lead to a race condition with other code that's locking
	// the same file (because fcntl locks are based on the underlying file and
	// not a particular file descriptor within a process). In fact, the Go
	// standard library and runtime do a similar thing with calls to close,
	// intentionally not handling EINTR from close due to a lack of information
	// about the state of the file descriptor and the desire to avoid a race
	// condition if that file descriptor is reused. In reality though, EINTR
	// here is exceedingly unlikely, and the usage of this code within Mutagen
	// is quite limited, with file unlocking only occurring right before a
	// process exits. So, for now, we'll just keep this consistent with the
	// locking case, but this EINTR behavior might be worth revisiting if it
	// becomes a problem.
	if err := fcntlFlockRetryingOnEINTR(l.file.Fd(), unix.F_SETLK, &unlockSpec); err != nil {
		return err
	}

	// Mark the lock as no longer being held.
	l.held = false

	// Success.
	return nil
}
