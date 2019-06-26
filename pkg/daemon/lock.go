package daemon

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem/locking"
)

const (
	// lockName is the name of the daemon lock file. It resides within the
	// daemon subdirectory of the Mutagen directory.
	lockName = "daemon.lock"
)

// AcquireLock is a convenience function which attempts to acquire the daemon
// lock and returns a locked file locker.
func AcquireLock() (*locking.Locker, error) {
	// Compute the lock path.
	lockPath, err := subpath(lockName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute daemon lock path")
	}

	// Create the daemon locker and attempt to acquire the lock.
	locker, err := locking.NewLocker(lockPath, 0600)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create daemon locker")
	} else if err = locker.Lock(false); err != nil {
		locker.Close()
		return nil, err
	}

	// Success.
	return locker, nil
}
