package state

import (
	"sync"
)

// Tracker provides index-based state tracking using a condition variable.
type Tracker struct {
	// change is the condition variable used to track changes.
	change *sync.Cond
	// index is the current state index.
	index uint64
	// poisoned indicates whether or not tracking has been terminated.
	poisoned bool
}

// NewTracker creates a new tracker instance with state index 1.
func NewTracker() *Tracker {
	return &Tracker{
		change: sync.NewCond(&sync.Mutex{}),
		index:  1,
	}
}

// Poison terminates tracking.
func (t *Tracker) Poison() {
	// Acquire the state lock and ensure its release.
	t.change.L.Lock()
	defer t.change.L.Unlock()

	// Mark the state as poisoned and broadcast the change.
	t.poisoned = true
	t.change.Broadcast()
}

// NotifyOfChange indicates the state index and notifies waiters.
func (t *Tracker) NotifyOfChange() {
	// Acquire the state lock and ensure its release.
	t.change.L.Lock()
	defer t.change.L.Unlock()

	// Increment the state index and broadcast changes.
	t.index += 1
	t.change.Broadcast()
}

// WaitForChange waits for a state index change from the previous index.
func (t *Tracker) WaitForChange(previousIndex uint64) (uint64, bool) {
	// Acquire the state lock and ensure its release.
	t.change.L.Lock()
	defer t.change.L.Unlock()

	// Wait for the state index to change and return the new index.
	for t.index == previousIndex && !t.poisoned {
		t.change.Wait()
	}
	return t.index, t.poisoned
}
