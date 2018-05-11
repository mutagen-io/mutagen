package daemon

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

const (
	lockName = "daemon.lock"
)

type Lock struct {
	// locker is the daemon file lock, uniquely held by a single daemon
	// instance. Because the locking semantics vary by platform, hosting
	// processes should only attempt to create a single daemon lock at a time.
	locker *filesystem.Locker
}

func AcquireLock() (*Lock, error) {
	// Compute the lock path.
	lockPath, err := subpath(lockName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute daemon lock path")
	}

	// Create the daemon locker and attempt to acquire the lock.
	locker, err := filesystem.NewLocker(lockPath, 0600)
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

func (l *Lock) Unlock() error {
	return l.locker.Unlock()
}
