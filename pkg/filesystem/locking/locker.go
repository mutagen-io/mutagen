package locking

import (
	"os"

	"github.com/pkg/errors"
)

// Locker provides file locking facilities.
type Locker struct {
	// file is the underlying file object that's locked.
	file *os.File
	// held indicates whether or not the lock is currently held.
	held bool
}

// NewLocker attempts to create a lock with the file at the specified path,
// creating the file if necessary. The lock is returned in an unlocked state.
func NewLocker(path string, permissions os.FileMode) (*Locker, error) {
	mode := os.O_RDWR | os.O_CREATE | os.O_APPEND
	if file, err := os.OpenFile(path, mode, permissions); err != nil {
		return nil, errors.Wrap(err, "unable to open lock file")
	} else {
		return &Locker{file: file}, nil
	}
}

// Held returns whether or not the lock is currently held.
func (l *Locker) Held() bool {
	return l.held
}

// Read implements io.Reader.Read on the underlying file, but errors if the lock
// is not currently held.
func (l *Locker) Read(buffer []byte) (int, error) {
	// Verify that the lock is held.
	if !l.held {
		return 0, errors.New("lock not held")
	}

	// Perform the read.
	return l.file.Read(buffer)
}

// Write implements io.Writer.Write on the underlying file, but errors if the
// lock is not currently held.
func (l *Locker) Write(buffer []byte) (int, error) {
	// Verify that the lock is held.
	if !l.held {
		return 0, errors.New("lock not held")
	}

	// Perform the write.
	return l.file.Write(buffer)
}

// Truncate implements file truncation for the underlying file, but errors if
// the lock is not currently held.
func (l *Locker) Truncate(size int64) error {
	// Verify that the lock is held.
	if !l.held {
		return errors.New("lock not held")
	}

	// Perform the truncation.
	return l.file.Truncate(size)
}

// Close closes the file underlying the locker. This will release any lock held
// on the file and disable future locking. On POSIX platforms, this also
// releases other locks held on the same file.
func (l *Locker) Close() error {
	return l.file.Close()
}
