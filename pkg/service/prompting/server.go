package prompting

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// Server provides an implementation of the Prompting service.
type Server struct{}

// NewServer creates a new prompt server.
func NewServer() *Server {
	return &Server{}
}

// asyncPromptResponse provides a structure for returning prompt results
// asynchronously, allowing prompting to be cancelled.
type asyncPromptResponse struct {
	// response is the response returned by the prompter.
	response string
	// error is the error returned by the prompter.
	error error
}

// Prompt facilitates prompting by registered prompters.
func (s *Server) Prompt(ctx context.Context, request *PromptRequest) (*PromptResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, errors.Wrap(err, "invalid prompt request")
	}

	// Perform prompting from the global registry asynchronously.
	asyncResponse := make(chan asyncPromptResponse, 1)
	go func() {
		response, err := prompting.Prompt(request.Prompter, request.Prompt)
		asyncResponse <- asyncPromptResponse{response, err}
	}()
	select {
	case <-ctx.Done():
		return nil, errors.New("prompting cancelled while waiting for response")
	case r := <-asyncResponse:
		if r.error != nil {
			return nil, errors.Wrap(r.error, "unable to prompt")
		} else {
			return &PromptResponse{Response: r.response}, nil
		}
	}
}
