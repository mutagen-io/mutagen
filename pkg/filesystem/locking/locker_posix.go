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

	// Attempt to perform locking.
	err := unix.FcntlFlock(l.file.Fd(), operation, &lockSpec)

	// Check for success and update the internal state.
	if err == nil {
		l.held = true
	}

	// Done.
	return err
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

	// Attempt to perform unlocking.
	err := unix.FcntlFlock(l.file.Fd(), unix.F_SETLK, &unlockSpec)

	// Check for success and update the internal state.
	if err == nil {
		l.held = false
	}

	// Done.
	return err
}
