package forwarding

import (
	"context"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
)

// Server provides an implementation of the Forwarding service.
type Server struct {
	// manager is the underlying session manager.
	manager *forwarding.Manager
}

// NewServer creates a new session server.
func NewServer(manager *forwarding.Manager) *Server {
	return &Server{
		manager: manager,
	}
}

// Create creates a new session.
func (s *Server) Create(ctx context.Context, request *CreateRequest) (*CreateResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid create request: %w", err)
	}

	// Perform creation.
	session, err := s.manager.Create(
		ctx,
		request.Specification.Source,
		request.Specification.Destination,
		request.Specification.Configuration,
		request.Specification.ConfigurationSource,
		request.Specification.ConfigurationDestination,
		request.Specification.Name,
		request.Specification.Labels,
		request.Specification.Paused,
		request.Prompter,
	)
	if err != nil {
		return nil, err
	}

	// Success.
	return &CreateResponse{Session: session}, nil
}

// List lists existing sessions.
func (s *Server) List(ctx context.Context, request *ListRequest) (*ListResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid list request: %w", err)
	}

	// Perform listing.
	stateIndex, states, err := s.manager.List(ctx, request.Selection, request.PreviousStateIndex)
	if err != nil {
		return nil, err
	}

	// Success.
	return &ListResponse{
		StateIndex:    stateIndex,
		SessionStates: states,
	}, nil
}

// Pause pauses existing sessions.
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

// Resume resumes existing sessions.
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

// Terminate terminates existing sessions.
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
