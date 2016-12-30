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

type TerminateRequest struct {
	Session string
}

type TerminateResponse struct{}
