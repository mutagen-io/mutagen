package daemon

import (
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/filesystem/locking"
)

// Lock represents the global daemon lock. It is held by a single daemon
// instance at a time.
type Lock struct {
	// locker is the underlying file locker.
	locker *locking.Locker
}

// AcquireLock attempts to acquire the global daemon lock.
func AcquireLock() (*Lock, error) {
	// Compute the lock path.
	lockPath, err := subpath(lockName)
	if err != nil {
		return nil, fmt.Errorf("unable to compute daemon lock path: %w", err)
	}

	// Create the locker and attempt to acquire the lock.
	locker, err := locking.NewLocker(lockPath, 0600)
	if err != nil {
		return nil, fmt.Errorf("unable to create daemon file locker: %w", err)
	} else if err = locker.Lock(false); err != nil {
		locker.Close()
		return nil, err
	}

	// Create the lock.
	return &Lock{
		locker: locker,
	}, nil
}

// Release releases the daemon lock.
func (l *Lock) Release() error {
	// Release the lock.
	if err := l.locker.Unlock(); err != nil {
		l.locker.Close()
		return err
	}

	// Close the locker.
	if err := l.locker.Close(); err != nil {
		fmt.Errorf("unable to close locker: %w", err)
	}

	// Success.
	return nil
}
