package session

import (
	"github.com/havoc-io/mutagen/pkg/url"
)

type PromptRequest struct {
	Done    bool
	Message string
	Prompt  string
}

type PromptResponse struct {
	Response string
}

type CreateRequest struct {
	Alpha   *url.URL
	Beta    *url.URL
	Ignores []string
}

type CreateResponse struct {
	Session string
}

type ListRequest struct {
	PreviousStateIndex uint64
	All                bool
	SessionQueries     []string
}

type ListResponse struct {
	StateIndex    uint64
	SessionStates []SessionState
}

type PauseRequest struct {
	All            bool
	SessionQueries []string
}

type PauseResponse struct{}

type ResumeRequest struct {
	All            bool
	SessionQueries []string
}

type ResumeResponse struct{}

type TerminateRequest struct {
	All            bool
	SessionQueries []string
}

type TerminateResponse struct{}
