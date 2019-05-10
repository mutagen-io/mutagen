// +build !windows,!plan9

// TODO: Figure out what to do for Plan 9. It doesn't support FcntlFlock at all,
// but we might be able to ~emulate it with os.O_EXCL, but that wouldn't allow
// us to automatically release locks if a process dies.

package locking

import (
	"os"

	"golang.org/x/sys/unix"
)

// Lock attempts to acquire the file lock.
func (l *Locker) Lock(block bool) error {
	lockSpec := unix.Flock_t{
		Type:   unix.F_WRLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}
	operation := unix.F_SETLK
	if block {
		operation = unix.F_SETLKW
	}
	return unix.FcntlFlock(l.file.Fd(), operation, &lockSpec)
}

// Unlock releases the file lock.
func (l *Locker) Unlock() error {
	unlockSpec := unix.Flock_t{
		Type:   unix.F_UNLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}
	return unix.FcntlFlock(l.file.Fd(), unix.F_SETLK, &unlockSpec)
}
