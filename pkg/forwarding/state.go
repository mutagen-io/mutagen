package forwarding

import (
	"errors"
	"fmt"
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
		return fmt.Errorf("invalid session: %w", err)
	}

	// Ensure that the connection counts are sane.
	if s.OpenConnections > s.TotalConnections {
		return errors.New("invalid connection counts")
	}

	// Success.
	return nil
}

// copy creates a static copy of the state, deep-copying any mutable members.
func (s *State) copy() *State {
	return &State{
		Session:              s.Session.copy(),
		Status:               s.Status,
		SourceConnected:      s.SourceConnected,
		DestinationConnected: s.DestinationConnected,
		LastError:            s.LastError,
		OpenConnections:      s.OpenConnections,
		TotalConnections:     s.TotalConnections,
	}
}
