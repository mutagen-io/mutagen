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
	AlphaChanges []Change `json:"alphaChanges"`
	// BetaChanges are the relevant changes on beta.
	BetaChanges []Change `json:"betaChanges"`
}

// loadFromInternal sets a conflict to match an internal Protocol Buffers
// representation. The conflict must be valid.
func (c *Conflict) loadFromInternal(conflict *core.Conflict) {
	// Propagate the conflict root.
	c.Root = conflict.Root

	// Propagate alpha changes.
	c.AlphaChanges = make([]Change, len(conflict.AlphaChanges))
	for i := 0; i < len(conflict.AlphaChanges); i++ {
		c.AlphaChanges[i].loadFromInternal(conflict.AlphaChanges[i])
	}

	// Propagate beta changes.
	c.BetaChanges = make([]Change, len(conflict.BetaChanges))
	for i := 0; i < len(conflict.BetaChanges); i++ {
		c.BetaChanges[i].loadFromInternal(conflict.BetaChanges[i])
	}
}

// exportConflicts is a convenience function that calls
// Conflict.loadFromInternal for a slice of conflicts.
func exportConflicts(conflicts []*core.Conflict) []Conflict {
	// If there are no conflicts, then just return a nil slice.
	count := len(conflicts)
	if count == 0 {
		return nil
	}

	// Create the resulting slice.
	results := make([]Conflict, count)
	for i := 0; i < count; i++ {
		results[i].loadFromInternal(conflicts[i])
	}

	// Done.
	return results
}
