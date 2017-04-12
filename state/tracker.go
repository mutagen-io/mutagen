package state

import (
	"sync"
)

type Tracker struct {
	change   *sync.Cond
	index    uint64
	poisoned bool
}

func NewTracker() *Tracker {
	return &Tracker{
		change: sync.NewCond(&sync.Mutex{}),
		index:  1,
	}
}

func (t *Tracker) Poison() {
	// Acquire the state lock and ensure its release.
	t.change.L.Lock()
	defer t.change.L.Unlock()

	// Mark the state as poisoned and broadcast the change.
	t.poisoned = true
	t.change.Broadcast()
}

func (t *Tracker) NotifyOfChange() {
	// Acquire the state lock and ensure its release.
	t.change.L.Lock()
	defer t.change.L.Unlock()

	// Increment the state index and broadcast changes.
	t.index += 1
	t.change.Broadcast()
}

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
