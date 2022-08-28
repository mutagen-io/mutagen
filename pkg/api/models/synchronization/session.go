package synchronization

import (
	"fmt"
	"time"

	"github.com/mutagen-io/mutagen/pkg/synchronization"
)

// Session represents a synchronization session.
type Session struct {
	// Identifier is the unique session identifier.
	Identifier string `json:"identifier"`
	// Version is the session version.
	Version synchronization.Version `json:"version"`
	// CreationTime is the session creation timestamp.
	CreationTime string `json:"creationTime"`
	// CreatingVersion is the version of Mutagen that created the session.
	CreatingVersion string `json:"creatingVersion"`
	// Alpha stores the alpha endpoint's configuration and state.
	Alpha Endpoint `json:"alpha"`
	// Beta stores the beta endpoint's configuration and state.
	Beta Endpoint `json:"beta"`
	// Configuration is the session configuration.
	Configuration
	// Name is the session name.
	Name string `json:"name,omitempty"`
	// Label are the session labels.
	Labels map[string]string `json:"labels,omitempty"`
	// Paused indicates whether or not the session is paused.
	Paused bool `json:"paused"`
	// Status is the session status.
	Status synchronization.Status `json:"status"`
	// SessionState stores state fields relevant to running sessions. It is
	// non-nil if and only if the session is unpaused.
	*SessionState
}

// SessionState encodes fields relevant to unpaused sessions.
type SessionState struct {
	// LastError is the last synchronization error to occur.
	LastError string `json:"lastError,omitempty"`
	// SuccessfulCycles is the number of successful synchronization cycles to
	// occur since successfully connecting to the endpoints.
	SuccessfulCycles uint64 `json:"successfulCycles"`
	// Conflicts are the conflicts that identified during reconciliation. This
	// list may be a truncated version of the full list if too many conflicts
	// are encountered to report via the API.
	Conflicts []Conflict `json:"conflicts,omitempty"`
	// ExcludedConflicts is the number of conflicts that have been excluded from
	// Conflicts due to truncation. This value can only be non-zero if conflicts
	// is non-empty.
	ExcludedConflicts uint64 `json:"excludedConflicts,omitempty"`
}

// loadFromInternal sets a session to match an internal Protocol Buffers session
// state representation. The session state must be valid.
func (s *Session) loadFromInternal(state *synchronization.State) {
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
	s.Status = state.Status

	// Propagate endpoint information.
	s.Alpha.loadFromInternal(
		state.Session.Alpha,
		state.Session.ConfigurationAlpha,
		state.AlphaState,
	)
	s.Beta.loadFromInternal(
		state.Session.Beta,
		state.Session.ConfigurationBeta,
		state.BetaState,
	)

	// Propagate configuration information.
	s.Configuration.loadFromInternal(state.Session.Configuration)

	// Propagate state information if the session isn't paused.
	if state.Session.Paused {
		s.SessionState = nil
	} else {
		s.SessionState = &SessionState{
			LastError:         state.LastError,
			SuccessfulCycles:  state.SuccessfulCycles,
			Conflicts:         exportConflicts(state.Conflicts),
			ExcludedConflicts: state.ExcludedConflicts,
		}
	}
}

// ExportSessions converts a slice of internal session state representations to
// a slice of public session representations. It is guaranteed to return a
// non-nil value, even in the case of an empty slice.
func ExportSessions(states []*synchronization.State) []Session {
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
