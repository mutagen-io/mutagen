package session

import (
	"github.com/havoc-io/mutagen/sync"
)

type SynchronizationStatus uint8

const (
	SynchronizationStatusDisconnected = iota
	SynchronizationStatusConnecting
	SynchronizationStatusInitializing
	SynchronizationStatusScanning
	SynchronizationStatusReconciling
	SynchronizationStatusStaging
	SynchronizationStatusTransitioning
	SynchronizationStatusSaving
)

func (s SynchronizationStatus) String() string {
	switch s {
	case SynchronizationStatusDisconnected:
		return "Disconnected"
	case SynchronizationStatusConnecting:
		return "Connecting to endpoints"
	case SynchronizationStatusInitializing:
		return "Initializing endpoints"
	case SynchronizationStatusScanning:
		return "Watching for changes"
	case SynchronizationStatusReconciling:
		return "Reconciling changes"
	case SynchronizationStatusStaging:
		return "Staging changes"
	case SynchronizationStatusTransitioning:
		return "Applying changes"
	case SynchronizationStatusSaving:
		return "Saving archive"
	default:
		return "Unknown"
	}
}

type StagingStatus struct {
	Path  string
	Index uint64
	Total uint64
}

// SynchronizationState represents the current state of a synchronization loop.
type SynchronizationState struct {
	Status         SynchronizationStatus
	AlphaConnected bool
	BetaConnected  bool
	LastError      string
	AlphaStaging   StagingStatus
	BetaStaging    StagingStatus
	Conflicts      []sync.Conflict
	AlphaProblems  []sync.Problem
	BetaProblems   []sync.Problem
}

type SessionState struct {
	Session *Session
	State   SynchronizationState
}
