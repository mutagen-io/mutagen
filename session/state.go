package session

import (
	"sync"
)

type stateTracker struct {
	change *sync.Cond
	index  uint64
}

func newStateTracker() *stateTracker {
	return &stateTracker{
		change: sync.NewCond(&sync.Mutex{}),
		index:  1,
	}
}

func (s *stateTracker) lock() {
	s.change.L.Lock()
}

// TODO: Document that callers should pass 0 if they have no previous state
// index.
func (s *stateTracker) waitForChangeAndLock(previousIndex uint64) uint64 {
	s.change.L.Lock()
	for s.index == previousIndex {
		s.change.Wait()
	}
	return s.index
}

func (s *stateTracker) unlock() {
	s.change.L.Unlock()
}

func (s *stateTracker) notifyOfChangesAndUnlock() {
	s.index += 1
	s.change.Broadcast()
	s.change.L.Unlock()
}
