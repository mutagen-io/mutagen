package session

import (
	"github.com/pkg/errors"
)

// createStreamPrompter implements Prompter on top of a Sessions_CreateServer
// stream.
type createStreamPrompter struct {
	// stream is the underlying Sessions_CreateServer stream.
	stream Sessions_CreateServer
}

// sendReceive performs a send/receive cycle by sending a CreateResponse and
// receiving a CreateRequest.
func (p *createStreamPrompter) sendReceive(request *CreateResponse) (*CreateRequest, error) {
	// Send the request.
	if err := p.stream.Send(request); err != nil {
		return nil, errors.Wrap(err, "unable to send prompt request")
	}

	// Receive the response.
	if response, err := p.stream.Recv(); err != nil {
		return nil, errors.Wrap(err, "unable to receive prompt response")
	} else {
		return response, nil
	}
}

// Message implements the Message method of Prompter.
func (p *createStreamPrompter) Message(message string) error {
	_, err := p.sendReceive(&CreateResponse{Message: message})
	return err
}

// Prompt implements the Prompt method of Prompter.
func (p *createStreamPrompter) Prompt(prompt string) (string, error) {
	if response, err := p.sendReceive(&CreateResponse{Prompt: prompt}); err != nil {
		return "", err
	} else {
		return response.Response, nil
	}
}

// resumeStreamPrompter implements Prompter on top of a Sessions_ResumeServer
// stream.
type resumeStreamPrompter struct {
	// stream is the underlying Sessions_ResumeServer stream.
	stream Sessions_ResumeServer
}

// sendReceive performs a send/receive cycle by sending a CreateResponse and
// receiving a CreateRequest.
func (p *resumeStreamPrompter) sendReceive(request *ResumeResponse) (*ResumeRequest, error) {
	// Send the request.
	if err := p.stream.Send(request); err != nil {
		return nil, errors.Wrap(err, "unable to send prompt request")
	}

	// Receive the response.
	if response, err := p.stream.Recv(); err != nil {
		return nil, errors.Wrap(err, "unable to receive prompt response")
	} else {
		return response, nil
	}
}

// Message implements the Message method of Prompter.
func (p *resumeStreamPrompter) Message(message string) error {
	_, err := p.sendReceive(&ResumeResponse{Message: message})
	return err
}

// Prompt implements the Prompt method of Prompter.
func (p *resumeStreamPrompter) Prompt(prompt string) (string, error) {
	if response, err := p.sendReceive(&ResumeResponse{Prompt: prompt}); err != nil {
		return "", err
	} else {
		return response.Response, nil
	}
}
