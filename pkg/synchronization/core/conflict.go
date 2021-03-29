package core

import (
	"errors"
	"fmt"
	"sort"
)

// EnsureValid ensures that Conflict's invariants are respected.
func (c *Conflict) EnsureValid() error {
	// A nil conflict is not valid.
	if c == nil {
		return errors.New("nil conflict")
	}

	// There's not much validation we can perform on the conflict root path. It
	// may be empty and any format validation would be limited.

	// Each side's changes must be non-empty and must all be valid. We always
	// allow unsynchronizable content in conflict changes because they can be
	// the source of the conflict.
	if len(c.AlphaChanges) == 0 {
		return errors.New("conflict has no changes to alpha")
	} else {
		for _, change := range c.AlphaChanges {
			if err := change.EnsureValid(false); err != nil {
				return fmt.Errorf("invalid alpha change detected: %w", err)
			}
		}
	}
	if len(c.BetaChanges) == 0 {
		return errors.New("conflict has no changes to beta")
	} else {
		for _, change := range c.BetaChanges {
			if err := change.EnsureValid(false); err != nil {
				return fmt.Errorf("invalid beta change detected: %w", err)
			}
		}
	}

	// There's technically a bit more validation we could do, but it would be
	// expensive and wouldn't be exhaustive in any case. The purpose of this
	// function is simply to enforce memory safety invariants and algorithmic
	// invariants. Memory safety is fully verified by the checks performed here,
	// and conflicts don't enter into the synchronization algorithm (they're
	// purely a byproduct of it), so they can't corrupt a synchronization
	// session (and they come from a trusted source anyway - the daemon's
	// reconciliation algorithm).

	// Success.
	return nil
}

// Slim returns a copy of the conflict where each Change object has had its root
// entry reduced to a shallow copy (i.e. excluding contents). The conflict will
// still have enough metadata to determine its root path, and it will be
// considered valid.
func (c *Conflict) Slim() *Conflict {
	// Recompute alpha changes.
	alphaChanges := make([]*Change, len(c.AlphaChanges))
	for a, change := range c.AlphaChanges {
		alphaChanges[a] = change.slim()
	}

	// Recompute beta changes.
	betaChanges := make([]*Change, len(c.BetaChanges))
	for b, change := range c.BetaChanges {
		betaChanges[b] = change.slim()
	}

	// Done.
	return &Conflict{
		Root:         c.Root,
		AlphaChanges: alphaChanges,
		BetaChanges:  betaChanges,
	}
}

// CopyConflicts creates a copy of a list of conflicts in a new slice, usually
// for the purpose of modifying the list. The conflict objects themselves are
// not copied. It preserves nil vs. non-nil characteristics for empty slices.
func CopyConflicts(conflicts []*Conflict) []*Conflict {
	// If the slice is nil, then preserve its nilness. For zero-length, non-nil
	// slices, we still allocate on the heap to preserve non-nilness.
	if conflicts == nil {
		return nil
	}

	// Make a copy.
	result := make([]*Conflict, len(conflicts))
	copy(result, conflicts)

	// Done.
	return result
}

// sortableConflictList implements sort.Interface for conflict lists.
type sortableConflictList []*Conflict

// Len implements sort.Interface.Len.
func (l sortableConflictList) Len() int {
	return len(l)
}

// Less implements sort.Interface.Less.
func (l sortableConflictList) Less(i, j int) bool {
	return pathLess(l[i].Root, l[j].Root)
}

// Swap implements sort.Interface.Swap.
func (l sortableConflictList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// SortConflicts sorts a list of conflicts based on their root conflict paths.
func SortConflicts(conflicts []*Conflict) {
	sort.Sort(sortableConflictList(conflicts))
}
