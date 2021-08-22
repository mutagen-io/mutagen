package prompting

import (
	"context"
	"errors"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/identifier"
	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// Server provides an implementation of the Prompting service.
type Server struct {
	// UnimplementedPromptingServer is the required base implementation.
	UnimplementedPromptingServer
}

// NewServer creates a new prompt server.
func NewServer() *Server {
	return &Server{}
}

// Host performs prompt hosting.
func (s *Server) Host(stream Prompting_HostServer) error {
	// Receive and validate the initial request.
	request, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("unable to receive initial request: %w", err)
	} else if err = request.ensureValid(hostRequestModeInitial); err != nil {
		return fmt.Errorf("received invalid initial request: %w", err)
	}

	// Create a unique identifier for the prompter.
	identifier, err := identifier.New(identifier.PrefixPrompter)
	if err != nil {
		return fmt.Errorf("unable to generate prompter identifier: %w", err)
	}

	// Send the initial response.
	if err := stream.Send(&HostResponse{Identifier: identifier}); err != nil {
		return fmt.Errorf("unable to send initial response: %w", err)
	}

	// Extract the request context.
	ctx := stream.Context()

	// Wrap the stream to create a prompter.
	prompter := &streamPrompter{
		allowPrompts: request.AllowPrompts,
		stream:       stream,
	}

	// Register the prompter.
	if err := prompting.RegisterPrompterWithIdentifier(identifier, prompter); err != nil {
		return fmt.Errorf("unable to register prompter: %w", err)
	}

	// Wait for the request or connection to be terminated.
	<-ctx.Done()

	// Unregister the promper.
	prompting.UnregisterPrompter(identifier)

	// Success.
	return nil
}

// asyncPromptResponse provides a structure for returning prompt results
// asynchronously, allowing prompting to be cancelled.
type asyncPromptResponse struct {
	// response is the response returned by the prompter.
	response string
	// error is the error returned by the prompter.
	error error
}

// Prompt performs prompting against registered prompters.
func (s *Server) Prompt(ctx context.Context, request *PromptRequest) (*PromptResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid prompt request: %w", err)
	}

	// Perform prompting from the global registry asynchronously.
	// TODO: Should we build cancellation into the Prompter interface itself?
	asyncResponse := make(chan asyncPromptResponse, 1)
	go func() {
		response, err := prompting.Prompt(request.Prompter, request.Prompt)
		asyncResponse <- asyncPromptResponse{response, err}
	}()

	// Wait for a response or cancellation.
	select {
	case <-ctx.Done():
		return nil, errors.New("prompting cancelled while waiting for response")
	case r := <-asyncResponse:
		if r.error != nil {
			return nil, fmt.Errorf("unable to prompt: %w", r.error)
		} else {
			return &PromptResponse{Response: r.response}, nil
		}
	}
}
