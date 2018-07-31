package session

import (
	"github.com/pkg/errors"

	promptpkg "github.com/havoc-io/mutagen/pkg/prompt"
)

type createStreamPrompter struct {
	stream Sessions_CreateServer
}

func (p *createStreamPrompter) Prompt(message, prompt string) (string, error) {
	// Create the request.
	request := &CreateResponse{
		Prompt: &promptpkg.Prompt{
			Message: message,
			Prompt:  prompt,
		},
	}

	// Send the request.
	if err := p.stream.Send(request); err != nil {
		return "", errors.Wrap(err, "unable to send prompt request")
	}

	// Receive the response.
	if response, err := p.stream.Recv(); err != nil {
		return "", errors.Wrap(err, "unable to receive prompt response")
	} else {
		return response.Response, nil
	}
}

type resumeStreamPrompter struct {
	stream Sessions_ResumeServer
}

func (p *resumeStreamPrompter) Prompt(message, prompt string) (string, error) {
	// Create the request.
	request := &ResumeResponse{
		Prompt: &promptpkg.Prompt{
			Message: message,
			Prompt:  prompt,
		},
	}

	// Send the request.
	if err := p.stream.Send(request); err != nil {
		return "", errors.Wrap(err, "unable to send prompt request")
	}

	// Receive the response.
	if response, err := p.stream.Recv(); err != nil {
		return "", errors.Wrap(err, "unable to receive prompt response")
	} else {
		return response.Response, nil
	}
}
