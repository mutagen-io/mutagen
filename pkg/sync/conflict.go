package sync

import (
	"github.com/pkg/errors"
)

// Root returns the root path for a conflict.
func (c *Conflict) Root() string {
	// Handle determination of the root path based on the number of changes on
	// each side. At least one of the sides should have exactly one change,
	// whose path will correspond to the conflict root. If both sides have
	// exactly one change, then their paths must be equal or one must be a
	// prefix (parent path) of the other, in which case the shorter path to be
	// the conflict root.
	if len(c.AlphaChanges) == 1 && len(c.BetaChanges) == 1 {
		if len(c.AlphaChanges[0].Path) < len(c.BetaChanges[0].Path) {
			return c.AlphaChanges[0].Path
		} else {
			return c.BetaChanges[0].Path
		}
	} else if len(c.AlphaChanges) == 1 && len(c.BetaChanges) != 1 {
		return c.AlphaChanges[0].Path
	} else if len(c.BetaChanges) == 1 && len(c.AlphaChanges) != 1 {
		return c.BetaChanges[0].Path
	} else {
		panic("invalid conflict")
	}
}

// EnsureValid ensures that Conflict's invariants are respected.
func (c *Conflict) EnsureValid() error {
	// A nil conflict is not valid.
	if c == nil {
		return errors.New("nil conflict")
	}

	// Each side's changes must be non-empty and must all be valid.
	if len(c.AlphaChanges) == 0 {
		return errors.New("conflict has no changes to alpha")
	} else {
		for _, change := range c.AlphaChanges {
			if err := change.EnsureValid(); err != nil {
				return errors.Wrap(err, "invalid alpha change detected")
			}
		}
	}
	if len(c.BetaChanges) == 0 {
		return errors.New("conflict has no changes to beta")
	} else {
		for _, change := range c.BetaChanges {
			if err := change.EnsureValid(); err != nil {
				return errors.Wrap(err, "invalid beta change detected")
			}
		}
	}

	// Ensure that at least one side has exactly one conflict. We know this must
	// be the case because at least one of the sides must have a change at the
	// root of the conflict.
	if len(c.AlphaChanges) != 1 && len(c.BetaChanges) != 1 {
		return errors.New("both sides of conflict have zero or multiple changes")
	}

	// There's technically a bit more validation we could do, e.g. ensuring that
	// each side uses a given path only once and that all of the paths on one of
	// the sides are subpaths of the single path on the other side (or that both
	// sides name the same path), but that becomes very complicated and
	// expensive, and it isn't really needed for memory safety purposes.

	// Success.
	return nil
}
