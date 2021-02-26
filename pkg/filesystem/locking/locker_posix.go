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

	// Attempt to perform unlocking. Unlike the locking case, we don't retry if
	// EINTR is encountered because we don't have any information about the
	// state of the lock in that case (POSIX doesn't even allow for EINTR when
	// using F_SETLK and the Linux documentation only covers the locking case).
	// If the lock was successfully unlocked and we retry due to EINTR, then we
	// might end up in a race condition with other code trying to acquire the
	// lock. This is the same issue as with calls to close returning EINTR, in
	// which case the Go standard library and runtime (and Mutagen) don't retry
	// the operation because it's safer to err on the side of failure.
	//
	// In any case, this isn't ever going to be an issue for Mutagen in practice
	// because (a) EINTR is exceedingly unlikely here, (b) this code is only
	// used in Mutagen right before a process exits (usually in a defer that
	// ignores errors), and (c) Mutagen already has to be careful to avoid
	// multiple code paths managing locks on the same file (because POSIX
	// releases all fcntl locks for a file if any file descriptor for that lock
	// in the process is closed).
	if err := unix.FcntlFlock(l.file.Fd(), unix.F_SETLK, &unlockSpec); err != nil {
		return err
	}

	// Mark the lock as no longer being held.
	l.held = false

	// Success.
	return nil
}
