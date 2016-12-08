package state

import (
	"sync"
)

type StateIndex uint64

type State struct {
	index  StateIndex
	change *sync.Cond
	parent *State
}

func NewState() *State {
	return &State{
		index:  StateIndex(1),
		change: sync.NewCond(&sync.Mutex{}),
	}
}

func (s *State) Substate() *State {
	return &State{
		index:  StateIndex(1),
		change: sync.NewCond(s.change.L),
		parent: s,
	}
}

func (s *State) Lock() {
	// Acquire the lock for the entire state hierarchy (they all share a lock).
	s.change.L.Lock()
}

func (s *State) incrementIndexRecursive() {
	// Increment our state index.
	s.index += 1

	// Recursively incrememnt the index of any parent states.
	if s.parent != nil {
		s.parent.incrementIndexRecursive()
	}
}

func (s *State) broadcastRecursive() {
	// Broadcast to any listeners watching this state.
	s.change.Broadcast()

	// Recursively broadcast to any listeners watching any parent states.
	if s.parent != nil {
		s.parent.broadcastRecursive()
	}
}

func (s *State) NotifyOfChangesAndUnlock() {
	// Incrememnt the index for the entire state hierarchy.
	s.incrementIndexRecursive()

	// Broadcast notifications up the state hierarchy.
	s.broadcastRecursive()

	// Release the lock for the entire state hierarchy (they all share a lock).
	s.change.L.Unlock()
}

// TODO: Document that callers should pass 0 if they have no previous state
// index.
func (s *State) WaitForChangeAndLock(previousIndex StateIndex) StateIndex {
	// Acquire the lock for the entire state hierarchy (they all share a lock).
	s.change.L.Lock()

	// Wait until there is a change to the index for this state. Any substates
	// that change will also recursively update their parent states' indexes.
	for s.index == previousIndex {
		s.change.Wait()
	}

	// Return the new index. We'll still own the lock at this point.
	return s.index
}

func (s *State) Unlock() {
	// Release the lock for the entire state hierarchy (they all share a lock).
	s.change.L.Unlock()
}
