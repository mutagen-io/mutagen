package core

import (
	"github.com/pkg/errors"
)

// EnsureValid ensures that Problem's invariants are respected.
func (p *Problem) EnsureValid() error {
	// A nil problem is not valid.
	if p == nil {
		return errors.New("nil problem")
	}

	// We intentionally don't check that error != "", because we take the error
	// message as its given to use by the system, and thus we don't really have
	// any control over what it says. The more important thing is that the
	// presence of a non-nil problem indicates that there was a non-nil error.

	// Success.
	return nil
}
