package tunneling

import (
	"context"
	"fmt"

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
func (s *Server) Create(ctx context.Context, request *CreateRequest) (*CreateResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid create request: %w", err)
	}

	// Perform creation.
	hostCredentials, err := s.manager.Create(
		ctx,
		request.Specification.Configuration,
		request.Specification.Name,
		request.Specification.Labels,
		request.Specification.Paused,
		request.Prompter,
	)
	if err != nil {
		return nil, err
	}

	// Success.
	return &CreateResponse{HostCredentials: hostCredentials}, nil
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
func (s *Server) Pause(ctx context.Context, request *PauseRequest) (*PauseResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid pause request: %w", err)
	}

	// Perform pausing.
	if err := s.manager.Pause(ctx, request.Selection, request.Prompter); err != nil {
		return nil, err
	}

	// Success.
	return &PauseResponse{}, nil
}

// Resume resumes existing tunnels.
func (s *Server) Resume(ctx context.Context, request *ResumeRequest) (*ResumeResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid resume request: %w", err)
	}

	// Perform resuming.
	if err := s.manager.Resume(ctx, request.Selection, request.Prompter); err != nil {
		return nil, err
	}

	// Success.
	return &ResumeResponse{}, nil
}

// Terminate terminates existing tunnels.
func (s *Server) Terminate(ctx context.Context, request *TerminateRequest) (*TerminateResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid terminate request: %w", err)
	}

	// Perform termination.
	if err := s.manager.Terminate(ctx, request.Selection, request.Prompter); err != nil {
		return nil, err
	}

	// Success.
	return &TerminateResponse{}, nil
}
