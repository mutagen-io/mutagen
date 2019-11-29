package tunneling

import (
	"errors"
)

// ensureValid ensures that InitializeRequestVersion1's invariants are
// respected.
func (r *InitializeRequestVersion1) ensureValid() error {
	// Ensure that the request is non-nil.
	if r == nil {
		return errors.New("nil request")
	}

	// There's nothing we can do the validate the client version, but there's
	// also nothing we need to do since we'll require a match later.

	// Ensure that the mode is non-empty. There's not much more we can do in the
	// way of validation without knowing the available modes of every agent
	// binary that we offer.
	if r.Mode == "" {
		return errors.New("empty mode")
	}

	// Success.
	return nil
}

// ensureValid ensures that InitializeResponseVersion1's invariants are
// respected.
func (r *InitializeResponseVersion1) ensureValid() error {
	// Ensure that the response is non-nil.
	if r == nil {
		return errors.New("nil response")
	}

	// There's no need to validate the error string - any value is valid.

	// Success.
	return nil
}
