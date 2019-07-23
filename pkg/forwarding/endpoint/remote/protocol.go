package remote

import (
	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// ensureValid ensures that InitializeForwardingRequest's invariants are respected.
func (r *InitializeForwardingRequest) ensureValid() error {
	// A nil request is invalid.
	if r == nil {
		return errors.New("nil request")
	}

	// Ensure that the session version is supported.
	if !r.Version.Supported() {
		return errors.New("unsupported session version")
	}

	// Ensure that the configuration is valid.
	if err := r.Configuration.EnsureValid(false); err != nil {
		return errors.Wrap(err, "invalid configuration")
	}

	// Enforce that protocol is non-empty and supported.
	if r.Protocol == "" {
		return errors.New("empty protocol")
	} else if !forwarding.IsValidProtocol(r.Protocol) {
		return errors.New("invalid protocol")
	}

	// Enforce that address is non-empty.
	if r.Address == "" {
		return errors.New("empty address")
	}

	// There's no verification to be performed on the listener field.

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
