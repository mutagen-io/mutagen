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
	// Alpha is the alpha endpoint URL.
	Alpha URL `json:"alpha"`
	// Beta is the beta endpoint URL.
	Beta URL `json:"beta"`
	// Configuration is the session configuration.
	Configuration
	// ConfigurationAlpha is the alpha endpoint configuration.
	ConfigurationAlpha Configuration `json:"configurationAlpha"`
	// ConfigurationBeta is the beta endpoint configuration.
	ConfigurationBeta Configuration `json:"configurationBeta"`
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

// State encodes fields relevant to unpaused sessions.
type State struct {
	// Status is the session status.
	Status synchronization.Status `json:"status"`
	// AlphaConnected indicates whether or not the alpha endpoint is connected.
	AlphaConnected bool `json:"alphaConnected"`
	// BetaConnected indicates whether or not the beta endpoint is connected.
	BetaConnected bool `json:"betaConnected"`
	// LastError is the last synchronization error to occur.
	LastError string `json:"lastError,omitempty"`
	// SuccessfulCycles is the number of successful synchronization cycles to
	// occur since successfully connecting to the endpoints.
	SuccessfulCycles uint64 `json:"successfulCycles,omitempty"`
	// StagingStatus is the rsync staging status.
	StagingStatus *ReceiverStatus `json:"stagingStatus,omitempty"`
	// AlphaScanProblems is the list of non-terminal problems encountered during
	// scanning on alpha. This list may be a truncated version of the full list
	// if too many problems are encountered to report via the API.
	AlphaScanProblems []Problem `json:"alphaScanProblems,omitempty"`
	// ExcludedAlphaScanProblems is the number of problems that have been
	// excluded from AlphaScanProblems due to truncation. This value can only be
	// non-zero if alphaScanProblems is non-empty.
	ExcludedAlphaScanProblems uint64 `json:"excludedAlphaScanProblems,omitempty"`
	// BetaScanProblems is the list of non-terminal problems encountered during
	// scanning on beta. This list may be a truncated version of the full list
	// if too many problems are encountered to report via the API.
	BetaScanProblems []Problem `json:"betaScanProblems,omitempty"`
	// ExcludedBetaScanProblems is the number of problems that have been
	// excluded from BetaScanProblems due to truncation. This value can only be
	// non-zero if betaScanProblems is non-empty.
	ExcludedBetaScanProblems uint64 `json:"excludedBetaScanProblems,omitempty"`
	// Conflicts are the conflicts that identified during reconciliation. This
	// list may be a truncated version of the full list if too many conflicts
	// are encountered to report via the API.
	Conflicts []Conflict `json:"conflicts,omitempty"`
	// ExcludedConflicts is the number of conflicts that have been excluded from
	// Conflicts due to truncation. This value can only be non-zero if conflicts
	// is non-empty.
	ExcludedConflicts uint64 `json:"excludedConflicts,omitempty"`
	// AlphaTransitionProblems is the list of non-terminal problems encountered
	// during transition operations on alpha. This list may be a truncated
	// version of the full list if too many problems are encountered to report
	// via the API.
	AlphaTransitionProblems []Problem `json:"alphaTransitionProblems,omitempty"`
	// ExcludedAlphaTransitionProblems is the number of problems that have been
	// excluded from AlphaTransitionProblems due to truncation. This value can
	// only be non-zero if alphaTransitionProblems is non-empty.
	ExcludedAlphaTransitionProblems uint64 `json:"excludedAlphaTransitionProblems,omitempty"`
	// BetaTransitionProblems is the list of non-terminal problems encountered
	// during transition operations on beta. This list may be a truncated
	// version of the full list if too many problems are encountered to report
	// via the API.
	BetaTransitionProblems []Problem `json:"betaTransitionProblems,omitempty"`
	// ExcludedBetaTransitionProblems is the number of problems that have been
	// excluded from BetaTransitionProblems due to truncation. This value can
	// only be non-zero if betaTransitionProblems is non-empty.
	ExcludedBetaTransitionProblems uint64 `json:"excludedBetaTransitionProblems,omitempty"`
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

	// Propagate endpoint information.
	s.Alpha.loadFromInternal(state.Session.Alpha)
	s.Beta.loadFromInternal(state.Session.Beta)

	// Propagate configuration information.
	s.Configuration.loadFromInternal(state.Session.Configuration)
	s.ConfigurationAlpha.loadFromInternal(state.Session.ConfigurationAlpha)
	s.ConfigurationBeta.loadFromInternal(state.Session.ConfigurationBeta)

	// Propagate state information if the session isn't paused.
	if state.Session.Paused {
		s.State = nil
	} else {
		s.State = &State{
			Status:                          state.Status,
			AlphaConnected:                  state.AlphaConnected,
			BetaConnected:                   state.BetaConnected,
			LastError:                       state.LastError,
			SuccessfulCycles:                state.SuccessfulCycles,
			StagingStatus:                   newReceiverStatusFromInternalReceiverStatus(state.StagingStatus),
			AlphaScanProblems:               exportProblems(state.AlphaScanProblems),
			ExcludedAlphaScanProblems:       state.ExcludedAlphaScanProblems,
			BetaScanProblems:                exportProblems(state.BetaScanProblems),
			ExcludedBetaScanProblems:        state.ExcludedBetaScanProblems,
			Conflicts:                       exportConflicts(state.Conflicts),
			ExcludedConflicts:               state.ExcludedConflicts,
			AlphaTransitionProblems:         exportProblems(state.AlphaTransitionProblems),
			ExcludedAlphaTransitionProblems: state.ExcludedAlphaTransitionProblems,
			BetaTransitionProblems:          exportProblems(state.BetaTransitionProblems),
			ExcludedBetaTransitionProblems:  state.ExcludedBetaTransitionProblems,
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
