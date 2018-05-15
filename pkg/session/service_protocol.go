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

type ListRequestKind uint8

const (
	ListRequestKindSingle ListRequestKind = iota
	ListRequestKindRepeated
	ListRequestKindRepeatedLatest
)

type ListRequest struct {
	Kind           ListRequestKind
	SessionQueries []string
}

type ListResponse struct {
	SessionStates []SessionState
}

type PauseRequest struct {
	SessionQuery string
}

type PauseResponse struct{}

type ResumeRequest struct {
	SessionQuery string
}

type ResumeResponse struct{}

type TerminateRequest struct {
	SessionQuery string
}

type TerminateResponse struct{}
