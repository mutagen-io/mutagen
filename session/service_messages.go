package session

import (
	"github.com/havoc-io/mutagen/url"
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
	Session string
	Monitor bool
}

type ListResponse struct {
	Sessions []SessionState
}

type PauseRequest struct {
	Session string
}

type PauseResponse struct{}

type ResumeRequest struct {
	Session string
}

type ResumeResponse struct{}

type TerminateRequest struct {
	Session string
}

type TerminateResponse struct{}
