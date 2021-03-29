package synchronization

import (
	"context"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/synchronization"
)

// Server provides an implementation of the Synchronization service.
type Server struct {
	// manager is the underlying session manager.
	manager *synchronization.Manager
}

// NewServer creates a new session server.
func NewServer(manager *synchronization.Manager) *Server {
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
		request.Specification.Alpha,
		request.Specification.Beta,
		request.Specification.Configuration,
		request.Specification.ConfigurationAlpha,
		request.Specification.ConfigurationBeta,
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

// List queries session status.
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

// Flush flushes sessions.
func (s *Server) Flush(ctx context.Context, request *FlushRequest) (*FlushResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid flush request: %w", err)
	}

	// Perform flushing.
	if err := s.manager.Flush(ctx, request.Selection, request.Prompter, request.SkipWait); err != nil {
		return nil, err
	}

	// Success.
	return &FlushResponse{}, nil
}

// Pause pauses sessions.
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

// Resume resumes sessions.
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

// Reset resets sessions.
func (s *Server) Reset(ctx context.Context, request *ResetRequest) (*ResetResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid reset request: %w", err)
	}

	// Perform resuming.
	if err := s.manager.Reset(ctx, request.Selection, request.Prompter); err != nil {
		return nil, err
	}

	// Success.
	return &ResetResponse{}, nil
}

// Terminate terminates sessions.
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
