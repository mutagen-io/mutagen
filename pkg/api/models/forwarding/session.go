package forwarding

import (
	"fmt"
	"time"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
)

// Session represents a forwarding session.
type Session struct {
	// Identifier is the unique session identifier.
	Identifier string `json:"identifier"`
	// Version is the session version.
	Version forwarding.Version `json:"version"`
	// CreationTime is the session creation timestamp.
	CreationTime string `json:"creationTime"`
	// CreatingVersion is the version of Mutagen that created the session.
	CreatingVersion string `json:"creatingVersion"`
	// Source is the source endpoint URL.
	Source URL `json:"source"`
	// Destination is the destination endpoint URL.
	Destination URL `json:"destination"`
	// Configuration is the session configuration.
	Configuration
	// ConfigurationSource is the source endpoint configuration.
	ConfigurationSource Configuration `json:"configurationSource"`
	// ConfigurationDestination is the destination endpoint configuration.
	ConfigurationDestination Configuration `json:"configurationDestination"`
	// Name is the session name.
	Name string `json:"name,omitempty"`
	// Label are the session labels.
	Labels map[string]string `json:"labels,omitempty"`
	// Paused indicates whether or not the session is paused.
	Paused bool `json:"paused"`
	// State stores state fields relevant to running sessions. It is non-nil if
	// and only if the session is unpaused.
	*State
}

// EndpointState encodes the current state of a forwarding endpoint.
type EndpointState struct {
	// Connected indicates whether or not the controller is currently connected
	// to the endpoint.
	Connected bool `json:"connected"`
}

// State encodes fields relevant to unpaused sessions.
type State struct {
	// Status is the session status.
	Status forwarding.Status `json:"status"`
	// LastError is the last forwarding error to occur.
	LastError string `json:"lastError,omitempty"`
	// OpenConnections is the number of connections currently open and being
	// forwarded.
	OpenConnections uint64 `json:"openConnections"`
	// TotalConnections is the number of total connections that have been opened
	// and forwarded (including those that are currently open).
	TotalConnections uint64 `json:"totalConnections"`
	// TotalOutboundData is the total amount of data (in bytes) that has been
	// transmitted from source to destination across all forwarded connections.
	TotalOutboundData uint64 `json:"totalOutboundData"`
	// TotalInboundData is the total amount of data (in bytes) that has been
	// transmitted from destination to source across all forwarded connections.
	TotalInboundData uint64 `json:"totalInboundData"`
	// SourceState encodes the state of the source endpoint.
	SourceState EndpointState `json:"sourceState"`
	// DestinationState encodes the state of the destination endpoint.
	DestinationState EndpointState `json:"destinationState"`
}

// loadFromInternal sets a session to match an internal Protocol Buffers session
// state representation. The session state must be valid.
func (s *Session) loadFromInternal(state *forwarding.State) {
	// Propagate basic information.
	s.Identifier = state.Session.Identifier
	s.Version = state.Session.Version
	s.CreationTime = state.Session.CreationTime.AsTime().Format(time.RFC3339Nano)
	s.CreatingVersion = fmt.Sprintf("%d.%d.%d",
		state.Session.CreatingVersionMajor,
		state.Session.CreatingVersionMinor,
		state.Session.CreatingVersionPatch,
	)
	s.Name = state.Session.Name
	s.Labels = state.Session.Labels
	s.Paused = state.Session.Paused

	// Propagate endpoint information.
	s.Source.loadFromInternal(state.Session.Source)
	s.Destination.loadFromInternal(state.Session.Destination)

	// Propagate configuration information.
	s.Configuration.loadFromInternal(state.Session.Configuration)
	s.ConfigurationSource.loadFromInternal(state.Session.ConfigurationSource)
	s.ConfigurationDestination.loadFromInternal(state.Session.ConfigurationDestination)

	// Propagate state information if the session isn't paused.
	if state.Session.Paused {
		s.State = nil
	} else {
		s.State = &State{
			Status:            state.Status,
			LastError:         state.LastError,
			OpenConnections:   state.OpenConnections,
			TotalConnections:  state.TotalConnections,
			TotalOutboundData: state.TotalOutboundData,
			TotalInboundData:  state.TotalInboundData,
			SourceState: EndpointState{
				Connected: state.SourceState.Connected,
			},
			DestinationState: EndpointState{
				Connected: state.DestinationState.Connected,
			},
		}
	}
}

// ExportSessions converts a slice of internal session state representations to
// a slice of public session representations. It is guaranteed to return a
// non-nil value, even in the case of an empty slice.
func ExportSessions(states []*forwarding.State) []Session {
	// Create the resulting slice.
	count := len(states)
	results := make([]Session, count)

	// Propagate session information
	for i := 0; i < count; i++ {
		results[i].loadFromInternal(states[i])
	}

	// Done.
	return results
}
