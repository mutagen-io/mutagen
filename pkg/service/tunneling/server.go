package tunneling

import (
	"context"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/prompting"
	"github.com/mutagen-io/mutagen/pkg/tunneling"
)

// Server provides an implementation of the Tunneling service.
type Server struct {
	// manager is the underlying tunnel manager.
	manager *tunneling.Manager
}

// NewServer creates a new tunnel server.
func NewServer(manager *tunneling.Manager) *Server {
	return &Server{
		manager: manager,
	}
}

// Create creates a new tunnel.
func (s *Server) Create(stream Tunneling_CreateServer) error {
	// Receive and validate the request.
	request, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("unable to receive request: %w", err)
	} else if err = request.ensureValid(true); err != nil {
		return fmt.Errorf("received invalid create request: %w", err)
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompting.RegisterPrompter(&createStreamPrompter{stream})
	if err != nil {
		return fmt.Errorf("unable to register prompter: %w", err)
	}

	// Perform creation.
	hostCredentials, err := s.manager.Create(
		stream.Context(),
		request.Specification.Configuration,
		request.Specification.Name,
		request.Specification.Labels,
		request.Specification.Paused,
		prompter,
	)

	// Unregister the prompter.
	prompting.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal completion.
	if err := stream.Send(&CreateResponse{HostCredentials: hostCredentials}); err != nil {
		return fmt.Errorf("unable to send response: %w", err)
	}

	// Success.
	return nil
}

// List lists existing tunnels.
func (s *Server) List(ctx context.Context, request *ListRequest) (*ListResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("received invalid list request: %w", err)
	}

	// Perform listing.
	stateIndex, states, err := s.manager.List(ctx, request.Selection, request.PreviousStateIndex)
	if err != nil {
		return nil, err
	}

	// Success.
	return &ListResponse{
		StateIndex:   stateIndex,
		TunnelStates: states,
	}, nil
}

// Pause pauses existing tunnels.
func (s *Server) Pause(stream Tunneling_PauseServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("unable to receive request: %w", err)
	} else if err = request.ensureValid(true); err != nil {
		return fmt.Errorf("received invalid pause request: %w", err)
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompting.RegisterPrompter(&pauseStreamPrompter{stream})
	if err != nil {
		return fmt.Errorf("unable to register prompter: %w", err)
	}

	// Perform termination.
	err = s.manager.Pause(stream.Context(), request.Selection, prompter)

	// Unregister the prompter.
	prompting.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal completion.
	if err := stream.Send(&PauseResponse{}); err != nil {
		return fmt.Errorf("unable to send response: %w", err)
	}

	// Success.
	return nil
}

// Resume resumes existing tunnels.
func (s *Server) Resume(stream Tunneling_ResumeServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("unable to receive request: %w", err)
	} else if err = request.ensureValid(true); err != nil {
		return fmt.Errorf("received invalid resume request: %w", err)
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompting.RegisterPrompter(&resumeStreamPrompter{stream})
	if err != nil {
		return fmt.Errorf("unable to register prompter: %w", err)
	}

	// Perform resuming.
	err = s.manager.Resume(stream.Context(), request.Selection, prompter)

	// Unregister the prompter.
	prompting.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal completion.
	if err := stream.Send(&ResumeResponse{}); err != nil {
		return fmt.Errorf("unable to send response: %w", err)
	}

	// Success.
	return nil
}

// Terminate terminates existing tunnels.
func (s *Server) Terminate(stream Tunneling_TerminateServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("unable to receive request: %w", err)
	} else if err = request.ensureValid(true); err != nil {
		return fmt.Errorf("received invalid terminate request: %w", err)
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompting.RegisterPrompter(&terminateStreamPrompter{stream})
	if err != nil {
		return fmt.Errorf("unable to register prompter: %w", err)
	}

	// Perform termination.
	err = s.manager.Terminate(stream.Context(), request.Selection, prompter)

	// Unregister the prompter.
	prompting.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal completion.
	if err := stream.Send(&TerminateResponse{}); err != nil {
		return fmt.Errorf("unable to send response: %w", err)
	}

	// Success.
	return nil
}
