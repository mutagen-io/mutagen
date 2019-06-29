package daemon

import (
	"testing"
)

// TestLockCycle tests an acquisition/release cycle of the daemon lock.
func TestLockCycle(t *testing.T) {
	// Attempt to acquire the daemon lock.
	lock, err := AcquireLock()
	if err != nil {
		t.Fatal("unable to acquire lock:", err)
	}

	// Release the lock.
	if err := lock.Release(); err != nil {
		t.Fatal("unable to release lock:", err)
	}
}
