package session

import (
	"github.com/havoc-io/mutagen/url"
)

type PromptRequest struct {
	Message string
	Prompt  string
}

type PromptResponse struct {
	Response string
}

type CreateRequest struct {
	Alpha    *url.URL
	Beta     *url.URL
	Response *PromptResponse
}

type CreateResponse struct {
	Challenge *PromptRequest
	Error     string
}

type ListRequest struct {
	PreviousStateIndex uint64
}

type SessionState struct {
	Session *Session
	State   SynchronizationState
}

type ListResponse struct {
	StateIndex uint64
	Sessions   []SessionState
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

type TerminateRequest struct {
	Session string
}

type TerminateResponse struct {
	Error string
}
