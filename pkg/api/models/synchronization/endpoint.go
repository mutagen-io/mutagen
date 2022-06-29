package synchronization

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// Endpoint represents a synchronization endpoint.
type Endpoint struct {
	// Protocol endpoint transport protocol.
	Protocol url.Protocol `json:"protocol"`
	// User is the endpoint user.
	User string `json:"user,omitempty"`
	// Host is the endpoint host.
	Host string `json:"host,omitempty"`
	// Port is the endpoint port.
	Port uint16 `json:"port,omitempty"`
	// Path is the synchronization root on the endpoint.
	Path string `json:"path"`
	// Environment is the environment variable map to use for the transport.
	Environment map[string]string `json:"environment,omitempty"`
	// Parameters is the parameter map to use for the transport.
	Parameters map[string]string `json:"parameters,omitempty"`
	// Configuration is the endpoint-specific configuration.
	Configuration
	// Connected indicates whether or not the controller is currently connected
	// to the endpoint.
	Connected bool `json:"connected"`
	// EndpointState stores state fields relevant to connected endpoints. It is
	// non-nil if and only if the endpoint is connected.
	*EndpointState
}

// EndpointState encodes the current state of a synchronization endpoint.
type EndpointState struct {
	// Scanned indicates whether or not at least one scan has been performed on
	// the endpoint.
	Scanned bool `json:"scanned"`
	// Directories is the number of synchronizable directory entries contained
	// in the last snapshot from the endpoint.
	Directories uint64 `json:"directories,omitempty"`
	// Files is the number of synchronizable file entries contained in the last
	// snapshot from the endpoint.
	Files uint64 `json:"files,omitempty"`
	// SymbolicLinks is the number of synchronizable symbolic link entries
	// contained in the last snapshot from the endpoint.
	SymbolicLinks uint64 `json:"symbolicLinks,omitempty"`
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

// loadFromInternal sets an Endpoint to match internal Protocol Buffers
// representations. All parameters must be valid.
func (e *Endpoint) loadFromInternal(url *url.URL, configuration *synchronization.Configuration, state *synchronization.EndpointState) {
	// Propagate URL parameters.
	e.Protocol = url.Protocol
	e.User = url.User
	e.Host = url.Host
	e.Port = uint16(url.Port)
	e.Path = url.Path
	e.Environment = url.Environment
	e.Parameters = url.Parameters

	// Propagate configuration.
	e.Configuration.loadFromInternal(configuration)

	// Propagate connectivity.
	e.Connected = state.Connected

	// Propagate other state fields.
	if !e.Connected {
		e.EndpointState = nil
	} else {
		e.EndpointState = &EndpointState{
			Scanned:                    state.Scanned,
			Directories:                state.Directories,
			Files:                      state.Files,
			SymbolicLinks:              state.SymbolicLinks,
			TotalFileSize:              state.TotalFileSize,
			ScanProblems:               exportProblems(state.ScanProblems),
			ExcludedScanProblems:       state.ExcludedScanProblems,
			TransitionProblems:         exportProblems(state.TransitionProblems),
			ExcludedTransitionProblems: state.ExcludedTransitionProblems,
			StagingProgress:            newReceiverStateFromInternalReceiverState(state.StagingProgress),
		}
	}
}
