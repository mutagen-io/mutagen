package session

import (
	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/sync"
)

type SynchronizationStatus uint8

const (
	SynchronizationStatusDisconnected = iota
	SynchronizationStatusHaltedOnRootDeletion
	SynchronizationStatusConnectingAlpha
	SynchronizationStatusConnectingBeta
	SynchronizationStatusWatching
	SynchronizationStatusScanningAlpha
	SynchronizationStatusScanningBeta
	SynchronizationStatusWaitingForRescan
	SynchronizationStatusReconciling
	SynchronizationStatusStagingAlpha
	SynchronizationStatusStagingBeta
	SynchronizationStatusTransitioningAlpha
	SynchronizationStatusTransitioningBeta
	SynchronizationStatusSaving
)

func (s SynchronizationStatus) String() string {
	switch s {
	case SynchronizationStatusDisconnected:
		return "Disconnected"
	case SynchronizationStatusHaltedOnRootDeletion:
		return "Halted due to root deletion"
	case SynchronizationStatusConnectingAlpha:
		return "Connecting to alpha"
	case SynchronizationStatusConnectingBeta:
		return "Connecting to beta"
	case SynchronizationStatusWatching:
		return "Watching for changes"
	case SynchronizationStatusScanningAlpha:
		return "Scanning files on alpha"
	case SynchronizationStatusScanningBeta:
		return "Scanning files on beta"
	case SynchronizationStatusWaitingForRescan:
		return "Waiting for rescan"
	case SynchronizationStatusReconciling:
		return "Reconciling changes"
	case SynchronizationStatusStagingAlpha:
		return "Staging files on alpha"
	case SynchronizationStatusStagingBeta:
		return "Staging files on beta"
	case SynchronizationStatusTransitioningAlpha:
		return "Applying changes on alpha"
	case SynchronizationStatusTransitioningBeta:
		return "Applying changes on beta"
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
	Staging        rsync.ReceivingStatus
	Conflicts      []sync.Conflict
	AlphaProblems  []sync.Problem
	BetaProblems   []sync.Problem
}

type SessionState struct {
	Session *Session
	State   SynchronizationState
}
