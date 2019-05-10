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

// Lock represents a daemon lock instance.
type Lock struct {
	// locker is the daemon file lock, uniquely held by a single daemon
	// instance. Because the locking semantics vary by platform, hosting
	// processes should only attempt to create a single daemon lock at a time.
	locker *locking.Locker
}

// AcquireLock attempts to acquire the daemon lock. It is the only way to
// acquire a Lock instance.
func AcquireLock() (*Lock, error) {
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
		return nil, err
	}

	// Create the lock.
	return &Lock{
		locker: locker,
	}, nil
}

// Unlock releases the daemon lock.
func (l *Lock) Unlock() error {
	return l.locker.Unlock()
}
