package session

import (
	"github.com/havoc-io/mutagen/sync"
	"github.com/havoc-io/mutagen/url"
)

type PromptRequest struct {
	Message string
	Prompt  string
}

type PromptResponse struct {
	Response string
}

type StartRequest struct {
	Alpha    *url.URL
	Beta     *url.URL
	Response *PromptResponse
}

type StartResponse struct {
	Challenge *PromptRequest
	Error     string
}

type ListRequest struct {
	PreviousStateIndex uint64
}

type SynchronizationStatus uint8

const (
	SynchronizationStatusIdle = iota
	SynchronizationStatusInitializingAlpha
	SynchronizationStatusInitializingBeta
	SynchronizationStatusScanning
	SynchronizationStatusReconciling
	SynchronizationStatusStagingAlphaToBeta
	SynchronizationStatusStagingBetaToAlpha
	SynchronizationStatusApplyingAlpha
	SynchronizationStatusApplyingBeta
	SynchronizationStatusSaving
	SynchronizationStatusUpdatingAlpha
	SynchronizationStatusUpdatingBeta
)

type SessionState struct {
	Session *Session
	// TODO: Do we want these?
	AlphaConnected bool
	BetaConnected  bool
	Status         SynchronizationStatus
	Message        string
	Conflicts      []*sync.Conflict
	Problems       []*sync.Problem
}

type ListResponse struct {
	StateIndex uint64
	Sessions   []*SessionState
}

type PauseRequest struct {
	Session string
}

type PauseResponse struct {
	Error string
}

type ResumeRequest struct {
	Session  string
	Response *PromptResponse
}

type ResumeResponse struct {
	Challenge *PromptRequest
	Error     string
}

type StopRequest struct {
	Session string
}

type StopResponse struct {
	Error string
}
