package core

import (
	"github.com/pkg/errors"
)

func (a *Archive) EnsureValid() error {
	// A nil archive is not valid.
	if a == nil {
		return errors.New("nil archive")
	}

	// Ensure that the archive root is valid.
	if err := a.Root.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid archive root")
	}

	// Success.
	return nil
}
