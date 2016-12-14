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

	// Wait for the state index to change and record the new index. I'm not sure
	// we really need to be recording the index in a separate variable, but it's
	// not clear if the defer statements will be executed before or after return
	// values are copied to their destination, so I think we should copy the
	// value first and then return it.
	// TODO: Investigate this, maybe we can just return n.index.
	for n.index == previousIndex {
		n.change.Wait()
	}
	index := n.index

	// Done.
	return index
}
