package sync

import (
	"github.com/pkg/errors"
)

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

	// There's technically a bit more validation we could do to ensure that each
	// side uses a given path only once and that all of the paths on one side
	// are subpaths of the other (or that both sides name the same path), but
	// that becomes very complicated and expensive, and it isn't really needed
	// for memory safety.

	// Success.
	return nil
}
