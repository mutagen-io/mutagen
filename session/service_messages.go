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
	Alpha          *url.URL
	Beta           *url.URL
	DefaultIgnores bool
	Ignores        []string
}

type ListRequest struct {
	PreviousStateIndex uint64
}

type ListResponse struct {
	StateIndex uint64
	Sessions   []SessionState
}

type PauseRequest struct {
	Session string
}

type PauseResponse struct{}

type ResumeRequest struct {
	Session string
}

type TerminateRequest struct {
	Session string
}

type TerminateResponse struct{}
