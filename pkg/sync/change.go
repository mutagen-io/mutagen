package sync

import (
	"github.com/pkg/errors"
)

func (c *Change) EnsureValid() error {
	// A nil change is not valid.
	if c == nil {
		return errors.New("nil change")
	}

	// Technically we could validate the path, but that's error prone,
	// expensive, and not really needed for memory safety.

	// A change isn't valid if its old and new values are the same.
	if c.New.Equal(c.Old) {
		return errors.New("change entries are equal")
	}

	// Success.
	return nil
}
