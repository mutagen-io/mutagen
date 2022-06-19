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

// EndpointState encodes the current state of a synchronization endpoint.
type EndpointState struct {
	// Connected indicates whether or not the controller is currently connected
	// to the endpoint.
	Connected bool `json:"connected"`
	// DirectoryCount is the number of synchronizable directory entries
	// contained in the last snapshot from the endpoint.
	DirectoryCount uint64 `json:"directoryCount,omitempty"`
	// FileCount is the number of synchronizable file entries contained in the
	// last snapshot from the endpoint.
	FileCount uint64 `json:"fileCount,omitempty"`
	// SymbolicLinkCount is the number of synchronizable symbolic link entries
	// contained in the last snapshot from the endpoint.
	SymbolicLinkCount uint64 `json:"symbolicLinkCount,omitempty"`
	// TotalFileSize is the total size of all synchronizable files referenced by
	// the last snapshot from the endpoint.
	TotalFileSize uint64 `json:"totalFileSize,omitempty"`
	// ScanProblems is the list of non-terminal problems encountered during the
	// last scanning operation on the endpoint. This list may be a truncated
	// version of the full list if too many problems are encountered to report
	// via the API, in which case ExcludedScanProblems will be non-zero.
	ScanProblems []Problem `json:"scanProblems,omitempty"`
	// ExcludedScanProblems is the number of problems that have been excluded
	// from ScanProblems due to truncation. This value can be non-zero only if
	// ScanProblems is non-empty.
	ExcludedScanProblems uint64 `json:"excludedScanProblems,omitempty"`
	// TransitionProblems is the list of non-terminal problems encountered
	// during the last transition operation on the endpoint. This list may be a
	// truncated version of the full list if too many problems are encountered
	// to report via the API, in which case ExcludedTransitionProblems will be
	// non-zero.
	TransitionProblems []Problem `json:"transitionProblems,omitempty"`
	// ExcludedTransitionProblems is the number of problems that have been
	// excluded from TransitionProblems due to truncation. This value can be
	// non-zero only if TransitionProblems is non-empty.
	ExcludedTransitionProblems uint64 `json:"excludedTransitionProblems,omitempty"`
	// StagingProgress is the rsync staging progress. It is non-nil if and only
	// if the endpoint is currently staging files.
	StagingProgress *ReceiverState `json:"stagingProgress,omitempty"`
}

// State encodes fields relevant to unpaused sessions.
type State struct {
	// Status is the session status.
	Status synchronization.Status `json:"status"`
	// LastError is the last synchronization error to occur.
	LastError string `json:"lastError,omitempty"`
	// SuccessfulCycles is the number of successful synchronization cycles to
	// occur since successfully connecting to the endpoints.
	SuccessfulCycles uint64 `json:"successfulCycles,omitempty"`
	// Conflicts are the conflicts that identified during reconciliation. This
	// list may be a truncated version of the full list if too many conflicts
	// are encountered to report via the API.
	Conflicts []Conflict `json:"conflicts,omitempty"`
	// ExcludedConflicts is the number of conflicts that have been excluded from
	// Conflicts due to truncation. This value can only be non-zero if conflicts
	// is non-empty.
	ExcludedConflicts uint64 `json:"excludedConflicts,omitempty"`
	// AlphaState encodes the state of the alpha endpoint.
	AlphaState EndpointState `json:"alphaState"`
	// BetaState encodes the state of the beta endpoint.
	BetaState EndpointState `json:"betaState"`
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
			Status:            state.Status,
			LastError:         state.LastError,
			SuccessfulCycles:  state.SuccessfulCycles,
			Conflicts:         exportConflicts(state.Conflicts),
			ExcludedConflicts: state.ExcludedConflicts,
			AlphaState: EndpointState{
				Connected:                  state.AlphaState.Connected,
				DirectoryCount:             state.AlphaState.DirectoryCount,
				FileCount:                  state.AlphaState.FileCount,
				SymbolicLinkCount:          state.AlphaState.SymbolicLinkCount,
				TotalFileSize:              state.AlphaState.TotalFileSize,
				ScanProblems:               exportProblems(state.AlphaState.ScanProblems),
				ExcludedScanProblems:       state.AlphaState.ExcludedScanProblems,
				TransitionProblems:         exportProblems(state.AlphaState.TransitionProblems),
				ExcludedTransitionProblems: state.AlphaState.ExcludedTransitionProblems,
				StagingProgress:            newReceiverStateFromInternalReceiverState(state.AlphaState.StagingProgress),
			},
			BetaState: EndpointState{
				Connected:                  state.BetaState.Connected,
				DirectoryCount:             state.BetaState.DirectoryCount,
				FileCount:                  state.BetaState.FileCount,
				SymbolicLinkCount:          state.BetaState.SymbolicLinkCount,
				TotalFileSize:              state.BetaState.TotalFileSize,
				ScanProblems:               exportProblems(state.BetaState.ScanProblems),
				ExcludedScanProblems:       state.BetaState.ExcludedScanProblems,
				TransitionProblems:         exportProblems(state.BetaState.TransitionProblems),
				ExcludedTransitionProblems: state.BetaState.ExcludedTransitionProblems,
				StagingProgress:            newReceiverStateFromInternalReceiverState(state.BetaState.StagingProgress),
			},
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
