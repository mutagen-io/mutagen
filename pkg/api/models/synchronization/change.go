package synchronization

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// Change represents a filesystem content change.
type Change struct {
	// Path is the path of the root of the change, relative to the
	// synchronization root.
	Path string `json:"path"`
	// Old represents the old filesystem hierarchy at the change path. It may be
	// nil if no content previously existed.
	Old *Entry `json:"old"`
	// New represents the new filesystem hierarchy at the change path. It may be
	// nil if content has been deleted.
	New *Entry `json:"new"`
}

// loadFromInternal sets a change to match an internal Protocol Buffers
// representation. The change must be valid.
func (c *Change) loadFromInternal(change *core.Change) {
	c.Path = change.Path
	c.Old = newEntryFromInternalEntry(change.Old)
	c.New = newEntryFromInternalEntry(change.New)
}
