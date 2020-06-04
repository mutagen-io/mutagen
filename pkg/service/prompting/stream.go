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
	_, err := p.sendReceive(&HostResponse{Message: message})
	return err
}

// Prompt implements the Prompt method of Prompter.
func (p *streamPrompter) Prompt(prompt string) (string, error) {
	// Check whether or not prompts are supported by this client.
	if !p.allowPrompts {
		return "", errors.New("prompter only supports messaging")
	}

	// Perform the exchange.
	if response, err := p.sendReceive(&HostResponse{IsPrompt: true, Message: prompt}); err != nil {
		return "", err
	} else {
		return response.Response, nil
	}
}
