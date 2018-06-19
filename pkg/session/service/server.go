package service

import (
	"context"

	"github.com/pkg/errors"

	promptsvcpkg "github.com/havoc-io/mutagen/pkg/prompt/service"
	"github.com/havoc-io/mutagen/pkg/session"
)

type Server struct {
	manager      *session.Manager
	promptServer *promptsvcpkg.Server
}

func New(promptServer *promptsvcpkg.Server) (*Server, error) {
	// Create the session manager.
	manager, err := session.NewManager()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create session manager")
	}

	// Create the server.
	return &Server{
		manager:      manager,
		promptServer: promptServer,
	}, nil
}

func (s *Server) Shutdown() {
	s.manager.Shutdown()
}

func (s *Server) Create(stream Session_CreateServer) error {
	// Receive and validate the request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.Alpha.EnsureValid(); err != nil {
		return errors.Wrap(err, "alpha URL invalid")
	} else if err = request.Beta.EnsureValid(); err != nil {
		return errors.Wrap(err, "beta URL invalid")
	} else if err = request.Configuration.EnsureValid(session.ConfigurationSourceCreate); err != nil {
		return errors.Wrap(err, "session configuration invalid")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := s.promptServer.RegisterPrompter(&createStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform creation.
	// TODO: Figure out a way to monitor for cancellation.
	session, err := s.manager.Create(
		request.Alpha,
		request.Beta,
		request.Configuration,
		prompter,
	)

	// Unregister the prompter.
	s.promptServer.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal creation completion.
	if err := stream.Send(&CreateResponse{Session: session}); err != nil {
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}

func (s *Server) List(_ context.Context, request *ListRequest) (*ListResponse, error) {
	// Perform listing.
	// TODO: Figure out a way to monitor for cancellation.
	stateIndex, states, err := s.manager.List(request.PreviousStateIndex, request.Specifications)
	if err != nil {
		return nil, err
	}

	// Success.
	return &ListResponse{
		StateIndex:    stateIndex,
		SessionStates: states,
	}, nil
}

func (s *Server) Pause(_ context.Context, request *PauseRequest) (*PauseResponse, error) {
	// Perform pausing.
	// TODO: Figure out a way to monitor for cancellation.
	if err := s.manager.Pause(request.Specifications); err != nil {
		return nil, err
	}

	// Success.
	return &PauseResponse{}, nil
}

func (s *Server) Resume(stream Session_ResumeServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := s.promptServer.RegisterPrompter(&resumeStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform resuming.
	// TODO: Figure out a way to monitor for cancellation.
	err = s.manager.Resume(request.Specifications, prompter)

	// Unregister the prompter.
	s.promptServer.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal resume completion.
	if err := stream.Send(&ResumeResponse{}); err != nil {
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}

func (s *Server) Terminate(_ context.Context, request *TerminateRequest) (*TerminateResponse, error) {
	// Perform termination.
	// TODO: Figure out a way to monitor for cancellation.
	if err := s.manager.Terminate(request.Specifications); err != nil {
		return nil, err
	}

	// Success.
	return &TerminateResponse{}, nil
}
