package daemon

import (
	"bytes"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/logging"
)

// TestLockCycle tests an acquisition/release cycle of the daemon lock.
func TestLockCycle(t *testing.T) {
	logger := logging.NewLogger(logging.LevelError, &bytes.Buffer{})

	// Attempt to acquire the daemon lock.
	lock, err := AcquireLock(logger)
	if err != nil {
		t.Fatal("unable to acquire lock:", err)
	}

	// Release the lock.
	if err := lock.Release(); err != nil {
		t.Fatal("unable to release lock:", err)
	}
}
