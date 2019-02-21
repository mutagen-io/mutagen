package session

import (
	"context"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/prompt"
	"github.com/havoc-io/mutagen/pkg/session"
)

// Server provides an implementation of the Sessions service.
type Server struct {
	// manager is the underlying session manager.
	manager *session.Manager
}

// New creates an instances of the sessions server.
func New() (*Server, error) {
	// Create the session manager.
	manager, err := session.NewManager()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create session manager")
	}

	// Create the server.
	return &Server{
		manager: manager,
	}, nil
}

// Shutdown gracefully shuts down server resources.
func (s *Server) Shutdown() {
	// Forward the shutdown request to the session manager.
	s.manager.Shutdown()
}

// Create creates a new session.
func (s *Server) Create(stream Sessions_CreateServer) error {
	// Receive and validate the request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.ensureValid(true); err != nil {
		return errors.Wrap(err, "received invalid create request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompt.RegisterPrompter(&createStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform creation.
	// TODO: Figure out a way to monitor for cancellation.
	session, err := s.manager.Create(
		request.Alpha,
		request.Beta,
		request.Configuration,
		request.ConfigurationAlpha,
		request.ConfigurationBeta,
		prompter,
	)

	// Unregister the prompter.
	prompt.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal completion.
	if err := stream.Send(&CreateResponse{Session: session}); err != nil {
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}

// List lists existing sessions.
func (s *Server) List(_ context.Context, request *ListRequest) (*ListResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, errors.Wrap(err, "received invalid list request")
	}

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

// Flush flushes existing sessions.
func (s *Server) Flush(stream Sessions_FlushServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.ensureValid(true); err != nil {
		return errors.Wrap(err, "received invalid flush request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompt.RegisterPrompter(&flushStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform flush.
	err = s.manager.Flush(request.Specifications, prompter, request.SkipWait, stream.Context())

	// Unregister the prompter.
	prompt.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal completion.
	if err := stream.Send(&FlushResponse{}); err != nil {
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}

// Pause pauses existing sessions.
func (s *Server) Pause(stream Sessions_PauseServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.ensureValid(true); err != nil {
		return errors.Wrap(err, "received invalid pause request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompt.RegisterPrompter(&pauseStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform termination.
	// TODO: Figure out a way to monitor for cancellation.
	err = s.manager.Pause(request.Specifications, prompter)

	// Unregister the prompter.
	prompt.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal completion.
	if err := stream.Send(&PauseResponse{}); err != nil {
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}

// Resume resumes existing sessions.
func (s *Server) Resume(stream Sessions_ResumeServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.ensureValid(true); err != nil {
		return errors.Wrap(err, "received invalid resume request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompt.RegisterPrompter(&resumeStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform resuming.
	// TODO: Figure out a way to monitor for cancellation.
	err = s.manager.Resume(request.Specifications, prompter)

	// Unregister the prompter.
	prompt.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal completion.
	if err := stream.Send(&ResumeResponse{}); err != nil {
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}

// Terminate terminates existing sessions.
func (s *Server) Terminate(stream Sessions_TerminateServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.ensureValid(true); err != nil {
		return errors.Wrap(err, "received invalid terminate request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompt.RegisterPrompter(&terminateStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform termination.
	// TODO: Figure out a way to monitor for cancellation.
	err = s.manager.Terminate(request.Specifications, prompter)

	// Unregister the prompter.
	prompt.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal completion.
	if err := stream.Send(&TerminateResponse{}); err != nil {
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}
