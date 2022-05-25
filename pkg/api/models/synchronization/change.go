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

// NewChangeFromInternalChange creates a new change representation from an
// internal Protocol Buffers representation. The change must be valid.
func NewChangeFromInternalChange(change *core.Change) *Change {
	return &Change{
		Path: change.Path,
		Old:  NewEntryFromInternalEntry(change.Old),
		New:  NewEntryFromInternalEntry(change.New),
	}
}
