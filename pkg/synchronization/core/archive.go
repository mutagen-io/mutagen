package core

import (
	"errors"
	"fmt"
)

// EnsureValid ensures that Archive's invariants are respected. If
// synchronizable is true, then unsynchronizable content in the archive will be
// considered invalid.
func (a *Archive) EnsureValid(synchronizable bool) error {
	// A nil archive is not valid.
	if a == nil {
		return errors.New("nil archive")
	}

	// Ensure that the archive content is valid.
	if err := a.Content.EnsureValid(synchronizable); err != nil {
		return fmt.Errorf("invalid archive content: %w", err)
	}

	// Success.
	return nil
}
