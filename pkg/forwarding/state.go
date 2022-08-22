package forwarding

import (
	"errors"
	"fmt"
)

// Description returns a human-readable description of the session status.
func (s Status) Description() string {
	switch s {
	case Status_Disconnected:
		return "Disconnected"
	case Status_ConnectingSource:
		return "Connecting to source"
	case Status_ConnectingDestination:
		return "Connecting to destination"
	case Status_ForwardingConnections:
		return "Forwarding"
	default:
		return "Unknown"
	}
}

// MarshalText implements encoding.TextMarshaler.MarshalText.
func (s Status) MarshalText() ([]byte, error) {
	var result string
	switch s {
	case Status_Disconnected:
		result = "disconnected"
	case Status_ConnectingSource:
		result = "connecting-source"
	case Status_ConnectingDestination:
		result = "connecting-destination"
	case Status_ForwardingConnections:
		result = "forwarding"
	default:
		result = "unknown"
	}
	return []byte(result), nil
}

// ensureValid ensures that EndpointState's invariants are respected.
func (s *EndpointState) ensureValid() error {
	// A nil endpoint state is not valid.
	if s == nil {
		return errors.New("nil state")
	}

	// We could perform additional validation based on the session status and
	// the endpoint connectivity, but it would be prohibitively complex, and all
	// we're really concerned about here is memory safety and other structural
	// invariants.

	// Success.
	return nil
}

// EnsureValid ensures that State's invariants are respected.
func (s *State) EnsureValid() error {
	// A nil state is not valid.
	if s == nil {
		return errors.New("nil state")
	}

	// We could perform additional validation based on the session status, but
	// it would be prohibitively complex, and all we're really concerned about
	// here is memory safety and other structural invariants.

	// Ensure the session is valid.
	if err := s.Session.EnsureValid(); err != nil {
		return fmt.Errorf("invalid session: %w", err)
	}

	// Ensure that the connection counts are sane.
	if s.OpenConnections > s.TotalConnections {
		return errors.New("invalid connection counts")
	}

	// Ensure that endpoint states are valid.
	if err := s.SourceState.ensureValid(); err != nil {
		return fmt.Errorf("invalid source endpoint state: %w", err)
	} else if err = s.DestinationState.ensureValid(); err != nil {
		return fmt.Errorf("invalid destination endpoint state: %w", err)
	}

	// Success.
	return nil
}
