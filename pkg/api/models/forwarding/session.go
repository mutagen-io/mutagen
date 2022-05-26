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

type State struct {
	// Status is the session status.
	Status forwarding.Status `json:"status"`
	// SourceConnected indicates whether or not the source endpoint is
	// connected.
	SourceConnected bool `json:"sourceConnected"`
	// DestinationConnected indicates whether or not the destination endpoint is
	// connected.
	DestinationConnected bool `json:"destinationConnected"`
	// LastError is the last forwarding error to occur.
	LastError string `json:"lastError,omitempty"`
	// OpenConnections is the number of connections currently open and being
	// forwarded.
	OpenConnections uint64 `json:"openConnections"`
	// TotalConnections is the number of total connections that have been opened
	// and forwarded (including those that are currently open).
	TotalConnections uint64 `json:"totalConnections"`
}

// NewSessionFromInternalState constructs a new session API model from an
// internal Protocol Buffers state representation. The state must be valid.
func NewSessionFromInternalState(state *forwarding.State) *Session {
	// Create the result and propagate basic information.
	result := &Session{
		Identifier:   state.Session.Identifier,
		Version:      state.Session.Version,
		CreationTime: state.Session.CreationTime.AsTime().Format(time.RFC3339Nano),
		CreatingVersion: fmt.Sprintf("%d.%d.%d",
			state.Session.CreatingVersionMajor,
			state.Session.CreatingVersionMinor,
			state.Session.CreatingVersionPatch,
		),
		Name:   state.Session.Name,
		Labels: state.Session.Labels,
		Paused: state.Session.Paused,
	}

	// Propagate endpoint information.
	result.Source.LoadFromInternalURL(state.Session.Source)
	result.Destination.LoadFromInternalURL(state.Session.Destination)

	// Propagate configuration information.
	result.Configuration.LoadFromInternalConfiguration(state.Session.Configuration)
	result.ConfigurationSource.LoadFromInternalConfiguration(state.Session.ConfigurationSource)
	result.ConfigurationDestination.LoadFromInternalConfiguration(state.Session.ConfigurationDestination)

	// Propagate state information if the session isn't paused.
	if !state.Session.Paused {
		result.State = &State{
			Status:               state.Status,
			SourceConnected:      state.SourceConnected,
			DestinationConnected: state.DestinationConnected,
			LastError:            state.LastError,
			OpenConnections:      state.OpenConnections,
			TotalConnections:     state.TotalConnections,
		}
	}

	// Done.
	return result
}

// NewSessionSliceFromInternalStateSlice is a convenience function that calls
// NewSessionFromInternalState for a slice of session states. It is guaranteed
// to return a non-nil value, even in the case of an empty slice.
func NewSessionSliceFromInternalStateSlice(states []*forwarding.State) []*Session {
	// If there are no sessions, then return an empty slice. Unlike our other
	// conversion methods, we return a non-nil value in this case because the
	// session slice will be used as the root of a templating context and we
	// don't want it to render as "null" for JSON (or similar).
	count := len(states)
	if count == 0 {
		return make([]*Session, 0)
	}

	// Create the resulting slice.
	result := make([]*Session, count)
	for i := 0; i < count; i++ {
		result[i] = NewSessionFromInternalState(states[i])
	}

	// Done.
	return result
}
