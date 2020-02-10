package forwarding

import (
	"context"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/prompting"
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
func (s *Server) Create(stream Forwarding_CreateServer) error {
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
		request.Specification.Source,
		request.Specification.Destination,
		request.Specification.Configuration,
		request.Specification.ConfigurationSource,
		request.Specification.ConfigurationDestination,
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

// List lists existing sessions.
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

// Pause pauses existing sessions.
func (s *Server) Pause(stream Forwarding_PauseServer) error {
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
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}

// Resume resumes existing sessions.
func (s *Server) Resume(stream Forwarding_ResumeServer) error {
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
		return errors.Wrap(err, "unable to send response")
	}

	// Success.
	return nil
}

// Terminate terminates existing sessions.
func (s *Server) Terminate(stream Forwarding_TerminateServer) error {
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
