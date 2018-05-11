// +build !windows,!plan9

// TODO: Figure out what to do for Plan 9. It doesn't support FcntlFlock at all,
// but we might be able to ~emulate it with os.O_EXCL, but that wouldn't allow
// us to automatically release locks if a process dies.

package filesystem

import (
	"os"
	"syscall"
)

func (l *Locker) Lock(block bool) error {
	lockSpec := syscall.Flock_t{
		Type:   syscall.F_WRLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}
	operation := syscall.F_SETLK
	if block {
		operation = syscall.F_SETLKW
	}
	return syscall.FcntlFlock(l.file.Fd(), operation, &lockSpec)
}

func (l *Locker) Unlock() error {
	unlockSpec := syscall.Flock_t{
		Type:   syscall.F_UNLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
	}
	return syscall.FcntlFlock(l.file.Fd(), syscall.F_SETLK, &unlockSpec)
}
