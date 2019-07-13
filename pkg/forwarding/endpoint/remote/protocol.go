package remote

import (
	"github.com/pkg/errors"
)

// ensureValid ensures that InitializeForwardingRequest's invariants are respected.
func (r *InitializeForwardingRequest) ensureValid() error {
	// A nil request is invalid.
	if r == nil {
		return errors.New("nil request")
	}

	// There's no verification to be performed on the listener field.

	// Enforce that protocol is non-empty.
	if r.Protocol == "" {
		return errors.New("empty protocol")
	}

	// Enforce that address is non-empty.
	if r.Address == "" {
		return errors.New("empty address")
	}

	// Success.
	return nil
}

// ensureValid ensures that InitializeForwardingResponse's invariants are respected.
func (r *InitializeForwardingResponse) ensureValid() error {
	// A nil response is invalid.
	if r == nil {
		return errors.New("nil response")
	}

	// There's no verification to be performed on the error message.

	// Success.
	return nil
}
