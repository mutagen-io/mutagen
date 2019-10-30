package tunneling

import (
	"errors"
	"fmt"
)

const (
	// tokenMask is used to replace tokens in tunnel objects when copying.
	tokenMask = "*******"
)

// Description returns a human-readable description of the tunnel status.
func (s Status) Description() string {
	switch s {
	case Status_Disconnected:
		return "Waiting to connect"
	case Status_Connecting:
		return "Connecting"
	case Status_Connected:
		return "Connected"
	default:
		return "Unknown"
	}
}

// EnsureValid ensures that State's invariants are respected.
func (s *State) EnsureValid() error {
	// A nil state is not valid.
	if s == nil {
		return errors.New("nil state")
	}

	// Ensure the tunnel is valid.
	if err := s.Tunnel.EnsureValid(); err != nil {
		return fmt.Errorf("invalid tunnel: %w", err)
	}

	// We intentionally don't validate the status because we'd have to maintain
	// a pretty large conditional or data structure and we only use it for
	// display anyway, where it'll just render as "Unknown" or similar if it's
	// not valid.

	// Ensure that the session counts are sane.
	if s.ActiveSessions > s.TotalSessions {
		return errors.New("invalid session counts")
	}

	// Success.
	return nil
}

// Copy creates a copy of the state, deep-copying those members which are
// mutable. It also masks any sensitive members (e.g. API tokens and signing
// keys) from the copied tunnel object.
func (s *State) Copy() *State {
	// Create a shallow copy of the state.
	result := &State{}
	*result = *s

	// Create a shallow copy of the Tunnel member, if present.
	if s.Tunnel != nil {
		result.Tunnel = &Tunnel{}
		*result.Tunnel = *s.Tunnel
		result.Tunnel.Token = tokenMask
		result.Tunnel.Secret = make([]byte, len(result.Tunnel.Secret))
	}

	// All other composite members are either immutable values or considered to
	// be immutable, so we don't need to copy them.

	// Done.
	return result
}
