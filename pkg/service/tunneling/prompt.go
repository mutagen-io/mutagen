package tunneling

import (
	"errors"
	"fmt"
)

// createStreamPrompter implements Prompter on top of a Tunneling_CreateServer
// stream.
type createStreamPrompter struct {
	// stream is the underlying Tunneling_CreateServer stream.
	stream Tunneling_CreateServer
}

// sendReceive performs a send/receive cycle by sending a CreateResponse and
// receiving a CreateRequest.
func (p *createStreamPrompter) sendReceive(request *CreateResponse) (*CreateRequest, error) {
	// Send the request.
	if err := p.stream.Send(request); err != nil {
		return nil, fmt.Errorf("unable to send request: %w", err)
	}

	// Receive the response.
	if response, err := p.stream.Recv(); err != nil {
		return nil, fmt.Errorf("unable to receive response: %w", err)
	} else if err = response.ensureValid(false); err != nil {
		return nil, fmt.Errorf("invalid response received: %w", err)
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

// pauseStreamPrompter implements Prompter on top of a Tunneling_PauseServer
// stream.
type pauseStreamPrompter struct {
	// stream is the underlying Tunneling_PauseServer stream.
	stream Tunneling_PauseServer
}

// sendReceive performs a send/receive cycle by sending a PauseResponse and
// receiving a PauseRequest.
func (p *pauseStreamPrompter) sendReceive(request *PauseResponse) (*PauseRequest, error) {
	// Send the request.
	if err := p.stream.Send(request); err != nil {
		return nil, fmt.Errorf("unable to send request: %w", err)
	}

	// Receive the response.
	if response, err := p.stream.Recv(); err != nil {
		return nil, fmt.Errorf("unable to receive response: %w", err)
	} else if err = response.ensureValid(false); err != nil {
		return nil, fmt.Errorf("invalid response received: %w", err)
	} else {
		return response, nil
	}
}

// Message implements the Message method of Prompter.
func (p *pauseStreamPrompter) Message(message string) error {
	_, err := p.sendReceive(&PauseResponse{Message: message})
	return err
}

// Prompt implements the Prompt method of Prompter.
func (p *pauseStreamPrompter) Prompt(_ string) (string, error) {
	return "", errors.New("prompting not supported on pause message streams")
}

// resumeStreamPrompter implements Prompter on top of a Tunneling_ResumeServer
// stream.
type resumeStreamPrompter struct {
	// stream is the underlying Tunneling_ResumeServer stream.
	stream Tunneling_ResumeServer
}

// sendReceive performs a send/receive cycle by sending a ResumeResponse and
// receiving a ResumeRequest.
func (p *resumeStreamPrompter) sendReceive(request *ResumeResponse) (*ResumeRequest, error) {
	// Send the request.
	if err := p.stream.Send(request); err != nil {
		return nil, fmt.Errorf("unable to send request: %w", err)
	}

	// Receive the response.
	if response, err := p.stream.Recv(); err != nil {
		return nil, fmt.Errorf("unable to receive response: %w", err)
	} else if err = response.ensureValid(false); err != nil {
		return nil, fmt.Errorf("invalid response received: %w", err)
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

// terminateStreamPrompter implements Prompter on top of a
// Tunneling_TerminateServer stream.
type terminateStreamPrompter struct {
	// stream is the underlying Tunneling_TerminateServer stream.
	stream Tunneling_TerminateServer
}

// sendReceive performs a send/receive cycle by sending a TerminateResponse and
// receiving a TerminateRequest.
func (p *terminateStreamPrompter) sendReceive(request *TerminateResponse) (*TerminateRequest, error) {
	// Send the request.
	if err := p.stream.Send(request); err != nil {
		return nil, fmt.Errorf("unable to send request: %w", err)
	}

	// Receive the response.
	if response, err := p.stream.Recv(); err != nil {
		return nil, fmt.Errorf("unable to receive response: %w", err)
	} else if err = response.ensureValid(false); err != nil {
		return nil, fmt.Errorf("invalid response received: %w", err)
	} else {
		return response, nil
	}
}

// Message implements the Message method of Prompter.
func (p *terminateStreamPrompter) Message(message string) error {
	_, err := p.sendReceive(&TerminateResponse{Message: message})
	return err
}

// Prompt implements the Prompt method of Prompter.
func (p *terminateStreamPrompter) Prompt(_ string) (string, error) {
	return "", errors.New("prompting not supported on terminate message streams")
}
