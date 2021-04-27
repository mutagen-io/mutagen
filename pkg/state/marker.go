package state

import (
	"sync/atomic"
)

// Marker is a utility type used to track if a condition has occurred. It is
// safe for concurrent usage and designed for usage on hot paths. The zero value
// of Marker is unmarked.
type Marker struct {
	// storage is the underlying marker storage.
	storage uint32
}

// Mark idempotently marks the marker.
func (m *Marker) Mark() {
	atomic.StoreUint32(&m.storage, 1)
}

// Marked returns whether or not the marker is marked.
func (m *Marker) Marked() bool {
	return atomic.LoadUint32(&m.storage) == 1
}
