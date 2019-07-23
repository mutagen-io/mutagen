package forwarding

import (
	"github.com/pkg/errors"
)

// Description returns a human-readable description of the session status.
func (s Status) Description() string {
	switch s {
	case Status_Disconnected:
		return "Waiting to connect"
	case Status_ConnectingSource:
		return "Connecting to source"
	case Status_ConnectingDestination:
		return "Connecting to destination"
	case Status_ForwardingConnections:
		return "Forwarding connections"
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

	// We intentionally don't validate the status because we'd have to maintain
	// a pretty large conditional or data structure and we only use it for
	// display anyway, where it'll just render as "Unknown" or similar if it's
	// not valid.

	// Ensure the session is valid.
	if err := s.Session.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid session")
	}

	// Ensure that the connection counts are sane.
	if s.OpenConnections > s.TotalConnections {
		return errors.New("invalid connection counts")
	}

	// Success.
	return nil
}

// Copy creates a copy of the state, deep-copying those members which are
// mutable.
func (s *State) Copy() *State {
	// Create a shallow copy of the state.
	result := &State{}
	*result = *s

	// Create a shallow copy of the Session member, if present.
	if s.Session != nil {
		result.Session = &Session{}
		*result.Session = *s.Session
	}

	// All other composite members are either immutable values or considered to
	// be immutable, so we don't need to copy them.

	// Done.
	return result
}
