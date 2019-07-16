package synchronization

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/prompt"
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
func (s *Server) Create(stream Synchronization_CreateServer) error {
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
		request.Specification.Alpha,
		request.Specification.Beta,
		request.Specification.Configuration,
		request.Specification.ConfigurationAlpha,
		request.Specification.ConfigurationBeta,
		request.Specification.Labels,
		request.Specification.Paused,
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
	stateIndex, states, err := s.manager.List(request.Selection, request.PreviousStateIndex)
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
func (s *Server) Flush(stream Synchronization_FlushServer) error {
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
	err = s.manager.Flush(request.Selection, prompter, request.SkipWait, stream.Context())

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
func (s *Server) Pause(stream Synchronization_PauseServer) error {
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
	err = s.manager.Pause(request.Selection, prompter)

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
func (s *Server) Resume(stream Synchronization_ResumeServer) error {
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
	err = s.manager.Resume(request.Selection, prompter)

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
func (s *Server) Terminate(stream Synchronization_TerminateServer) error {
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
	err = s.manager.Terminate(request.Selection, prompter)

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
