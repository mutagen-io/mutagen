package synchronization

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// Conflict represents a filesystem change conflict.
type Conflict struct {
	// Root is the root path for the conflict, relative to the synchronization
	// root.
	Root string `json:"root"`
	// AlphaChanges are the relevant changes on alpha.
	AlphaChanges []*Change `json:"alphaChanges"`
	// BetaChanges are the relevant changes on beta.
	BetaChanges []*Change `json:"betaChanges"`
}

// NewConflictFromInternalConflict creates a new conflict representation from an
// internal Protocol Buffers representation. The conflict must be valid.
func NewConflictFromInternalConflict(conflict *core.Conflict) *Conflict {
	// Create the new conflict.
	result := &Conflict{
		Root: conflict.Root,
	}

	// Propagate alpha changes.
	result.AlphaChanges = make([]*Change, len(conflict.AlphaChanges))
	for i := 0; i < len(conflict.AlphaChanges); i++ {
		result.AlphaChanges[i] = NewChangeFromInternalChange(conflict.AlphaChanges[i])
	}

	// Propagate beta changes.
	result.BetaChanges = make([]*Change, len(conflict.BetaChanges))
	for i := 0; i < len(conflict.BetaChanges); i++ {
		result.BetaChanges[i] = NewChangeFromInternalChange(conflict.BetaChanges[i])
	}

	// Done.
	return result
}

// NewConflictSliceFromInternalConflictSlice is a convenience function that
// calls NewConflictFromInternalConflict for a slice of conflicts.
func NewConflictSliceFromInternalConflictSlice(conflicts []*core.Conflict) []*Conflict {
	// If there are no conflicts, then just return a nil slice.
	count := len(conflicts)
	if count == 0 {
		return nil
	}

	// Create the resulting slice.
	result := make([]*Conflict, count)
	for i := 0; i < count; i++ {
		result[i] = NewConflictFromInternalConflict(conflicts[i])
	}

	// Done.
	return result
}
