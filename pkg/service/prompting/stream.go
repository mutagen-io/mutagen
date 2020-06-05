package prompting

import (
	"errors"
	"fmt"
)

// streamPrompter implements Prompter on top of a Prompting_HostServer stream.
type streamPrompter struct {
	// allowPrompts indicates whether or not the client allows prompts.
	allowPrompts bool
	// stream is the underlying Prompting_HostServer stream.
	stream Prompting_HostServer
	// errored indicates whether or not the stream has encountered an error.
	errored bool
}

// sendReceive performs a send/receive cycle by sending a HostResponse and
// receiving a HostRequest.
func (p *streamPrompter) sendReceive(response *HostResponse) (*HostRequest, error) {
	// Send the request.
	if err := p.stream.Send(response); err != nil {
		return nil, fmt.Errorf("unable to send response: %w", err)
	}

	// Determine the expected request mode.
	mode := hostRequestModeMessageResponse
	if response.IsPrompt {
		mode = hostRequestModePromptResponse
	}

	// Receive the response.
	if request, err := p.stream.Recv(); err != nil {
		return nil, fmt.Errorf("unable to receive request: %w", err)
	} else if err = request.ensureValid(mode); err != nil {
		return nil, fmt.Errorf("invalid request received: %w", err)
	} else {
		return request, nil
	}
}

// Message implements the Message method of Prompter.
func (p *streamPrompter) Message(message string) error {
	// Check if a previous transmission error has occurred.
	if p.errored {
		return errors.New("prompter encountered previous error")
	}

	// Otherwise perform the messaging operation.
	if _, err := p.sendReceive(&HostResponse{Message: message}); err != nil {
		p.errored = true
		return err
	}

	// Success.
	return nil
}

// Prompt implements the Prompt method of Prompter.
func (p *streamPrompter) Prompt(prompt string) (string, error) {
	// Check if a previous transmission error has occurred.
	if p.errored {
		return "", errors.New("prompter encountered previous error")
	}

	// Check whether or not prompts are supported by this client.
	if !p.allowPrompts {
		return "", errors.New("prompter only supports messaging")
	}

	// Perform the exchange.
	response, err := p.sendReceive(&HostResponse{IsPrompt: true, Message: prompt})
	if err != nil {
		p.errored = true
		return "", err
	}

	// Success.
	return response.Response, nil
}
