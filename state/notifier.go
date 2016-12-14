package state

import (
	"sync"
)

type Notifier struct {
	change *sync.Cond
	index  uint64
}

func NewNotifier() *Notifier {
	return &Notifier{
		change: sync.NewCond(&sync.Mutex{}),
		index:  1,
	}
}

func (n *Notifier) Notify() {
	// Acquire the state lock and ensure its release.
	n.change.L.Lock()
	defer n.change.L.Unlock()

	// Increment the state index and broadcast changes.
	n.index += 1
	n.change.Broadcast()
}

func (n *Notifier) WaitForChange(previousIndex uint64) uint64 {
	// Acquire the state lock and ensure its release.
	n.change.L.Lock()
	defer n.change.L.Unlock()

	// Wait for the state index to change and return the new index.
	for n.index == previousIndex {
		n.change.Wait()
	}
	return n.index
}
