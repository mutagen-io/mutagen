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
	Alpha *URL `json:"alpha"`
	// Beta is the beta endpoint URL.
	Beta *URL `json:"beta"`
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
	AlphaScanProblems []*Problem `json:"alphaScanProblems,omitempty"`
	// ExcludedAlphaScanProblems is the number of problems that have been
	// excluded from AlphaScanProblems due to truncation. This value can only be
	// non-zero if alphaScanProblems is non-empty.
	ExcludedAlphaScanProblems uint64 `json:"excludedAlphaScanProblems,omitempty"`
	// BetaScanProblems is the list of non-terminal problems encountered during
	// scanning on beta. This list may be a truncated version of the full list
	// if too many problems are encountered to report via the API.
	BetaScanProblems []*Problem `json:"betaScanProblems,omitempty"`
	// ExcludedBetaScanProblems is the number of problems that have been
	// excluded from BetaScanProblems due to truncation. This value can only be
	// non-zero if betaScanProblems is non-empty.
	ExcludedBetaScanProblems uint64 `json:"excludedBetaScanProblems,omitempty"`
	// Conflicts are the conflicts that identified during reconciliation. This
	// list may be a truncated version of the full list if too many conflicts
	// are encountered to report via the API.
	Conflicts []*Conflict `json:"conflicts,omitempty"`
	// ExcludedConflicts is the number of conflicts that have been excluded from
	// Conflicts due to truncation. This value can only be non-zero if conflicts
	// is non-empty.
	ExcludedConflicts uint64 `json:"excludedConflicts,omitempty"`
	// AlphaTransitionProblems is the list of non-terminal problems encountered
	// during transition operations on alpha. This list may be a truncated
	// version of the full list if too many problems are encountered to report
	// via the API.
	AlphaTransitionProblems []*Problem `json:"alphaTransitionProblems,omitempty"`
	// ExcludedAlphaTransitionProblems is the number of problems that have been
	// excluded from AlphaTransitionProblems due to truncation. This value can
	// only be non-zero if alphaTransitionProblems is non-empty.
	ExcludedAlphaTransitionProblems uint64 `json:"excludedAlphaTransitionProblems,omitempty"`
	// BetaTransitionProblems is the list of non-terminal problems encountered
	// during transition operations on beta. This list may be a truncated
	// version of the full list if too many problems are encountered to report
	// via the API.
	BetaTransitionProblems []*Problem `json:"betaTransitionProblems,omitempty"`
	// ExcludedBetaTransitionProblems is the number of problems that have been
	// excluded from BetaTransitionProblems due to truncation. This value can
	// only be non-zero if betaTransitionProblems is non-empty.
	ExcludedBetaTransitionProblems uint64 `json:"excludedBetaTransitionProblems,omitempty"`
}

// NewSessionFromInternalState creates a new session representation from an
// internal Protocol Buffers representation. The session state must be valid.
func NewSessionFromInternalState(state *synchronization.State) *Session {
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
		Alpha:  NewURLFromInternalURL(state.Session.Alpha),
		Beta:   NewURLFromInternalURL(state.Session.Beta),
		Name:   state.Session.Name,
		Labels: state.Session.Labels,
		Paused: state.Session.Paused,
	}

	// Propagate configuration information.
	result.Configuration.LoadFromInternalConfiguration(state.Session.Configuration)
	result.ConfigurationAlpha.LoadFromInternalConfiguration(state.Session.ConfigurationAlpha)
	result.ConfigurationBeta.LoadFromInternalConfiguration(state.Session.ConfigurationBeta)

	// Propagate state information if the session isn't paused.
	if !state.Session.Paused {
		result.State = &State{
			Status:                          state.Status,
			AlphaConnected:                  state.AlphaConnected,
			BetaConnected:                   state.BetaConnected,
			LastError:                       state.LastError,
			SuccessfulCycles:                state.SuccessfulCycles,
			StagingStatus:                   NewReceiverStatusFromInternalReceiverStatus(state.StagingStatus),
			AlphaScanProblems:               NewProblemSliceFromInternalProblemSlice(state.AlphaScanProblems),
			ExcludedAlphaScanProblems:       state.ExcludedAlphaScanProblems,
			BetaScanProblems:                NewProblemSliceFromInternalProblemSlice(state.BetaScanProblems),
			ExcludedBetaScanProblems:        state.ExcludedBetaScanProblems,
			Conflicts:                       NewConflictSliceFromInternalConflictSlice(state.Conflicts),
			ExcludedConflicts:               state.ExcludedConflicts,
			AlphaTransitionProblems:         NewProblemSliceFromInternalProblemSlice(state.AlphaTransitionProblems),
			ExcludedAlphaTransitionProblems: state.ExcludedAlphaTransitionProblems,
			BetaTransitionProblems:          NewProblemSliceFromInternalProblemSlice(state.BetaTransitionProblems),
			ExcludedBetaTransitionProblems:  state.ExcludedBetaTransitionProblems,
		}
	}

	// Done.
	return result
}

// NewSessionSliceFromInternalStateSlice is a convenience function that calls
// NewSessionFromInternalState for a slice of session states. It is guaranteed
// to return a non-nil value, even in the case of an empty slice.
func NewSessionSliceFromInternalStateSlice(states []*synchronization.State) []*Session {
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
