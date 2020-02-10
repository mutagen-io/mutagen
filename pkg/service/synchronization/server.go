package synchronization

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/prompting"
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
	prompter, err := prompting.RegisterPrompter(&createStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform creation.
	session, err := s.manager.Create(
		stream.Context(),
		request.Specification.Alpha,
		request.Specification.Beta,
		request.Specification.Configuration,
		request.Specification.ConfigurationAlpha,
		request.Specification.ConfigurationBeta,
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
	if err := stream.Send(&CreateResponse{Session: session}); err != nil {
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}

// List queries session status.
func (s *Server) List(ctx context.Context, request *ListRequest) (*ListResponse, error) {
	// Validate the request.
	if err := request.ensureValid(); err != nil {
		return nil, errors.Wrap(err, "received invalid list request")
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
func (s *Server) Flush(stream Synchronization_FlushServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.ensureValid(true); err != nil {
		return errors.Wrap(err, "received invalid flush request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompting.RegisterPrompter(&flushStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform flush(es).
	err = s.manager.Flush(stream.Context(), request.Selection, prompter, request.SkipWait)

	// Unregister the prompter.
	prompting.UnregisterPrompter(prompter)

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

// Pause pauses sessions.
func (s *Server) Pause(stream Synchronization_PauseServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.ensureValid(true); err != nil {
		return errors.Wrap(err, "received invalid pause request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompting.RegisterPrompter(&pauseStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform pause(s).
	err = s.manager.Pause(stream.Context(), request.Selection, prompter)

	// Unregister the prompter.
	prompting.UnregisterPrompter(prompter)

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

// Resume resumes sessions.
func (s *Server) Resume(stream Synchronization_ResumeServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.ensureValid(true); err != nil {
		return errors.Wrap(err, "received invalid resume request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompting.RegisterPrompter(&resumeStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform resume(s).
	err = s.manager.Resume(stream.Context(), request.Selection, prompter)

	// Unregister the prompter.
	prompting.UnregisterPrompter(prompter)

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

// Reset resets sessions.
func (s *Server) Reset(stream Synchronization_ResetServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.ensureValid(true); err != nil {
		return errors.Wrap(err, "received invalid reset request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompting.RegisterPrompter(&resetStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform reset(s).
	err = s.manager.Reset(stream.Context(), request.Selection, prompter)

	// Unregister the prompter.
	prompting.UnregisterPrompter(prompter)

	// Handle any errors.
	if err != nil {
		return err
	}

	// Signal completion.
	if err := stream.Send(&ResetResponse{}); err != nil {
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}

// Terminate terminates sessions.
func (s *Server) Terminate(stream Synchronization_TerminateServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.ensureValid(true); err != nil {
		return errors.Wrap(err, "received invalid terminate request")
	}

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := prompting.RegisterPrompter(&terminateStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
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
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}
