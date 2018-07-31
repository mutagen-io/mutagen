package session

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	promptsvc "github.com/havoc-io/mutagen/pkg/prompt/service"
	"github.com/havoc-io/mutagen/pkg/session"
)

// Server provides an implementation of the Sessions service, providing methods
// for managing sessions. This Server is designed to operate as a singleton and
// can be accessed via the global DefaultServer variable.
type Server struct {
	// manager is the underlying session manager.
	manager *session.Manager
}

// defaultServerLock controls access to the defaultServer variable.
var defaultServerLock sync.RWMutex

// defaultServer is the default sessions server.
var defaultServer *Server

// DefaultServer provides the default sessions server, creating it if necessary.
func DefaultServer() (*Server, error) {
	// Optimistically attempt to grab the server.
	defaultServerLock.RLock()
	if defaultServer != nil {
		defer defaultServerLock.RUnlock()
		return defaultServer, nil
	}
	defaultServerLock.RUnlock()

	// Otherwise we need to create the server, so we'll need to get a write
	// lock on the server.
	defaultServerLock.Lock()
	defer defaultServerLock.Unlock()

	// It's possible that the server was created by someone else between our two
	// lockings, so see if we can just return it.
	if defaultServer != nil {
		return defaultServer, nil
	}

	// Create the default session manager.
	manager, err := session.NewManager()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create session manager")
	}

	// Create the default sessions server.
	defaultServer = &Server{
		manager: manager,
	}

	// Done.
	return defaultServer, nil
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
	} else if err = request.Alpha.EnsureValid(); err != nil {
		return errors.Wrap(err, "alpha URL invalid")
	} else if err = request.Beta.EnsureValid(); err != nil {
		return errors.Wrap(err, "beta URL invalid")
	} else if err = request.Configuration.EnsureValid(session.ConfigurationSourceCreate); err != nil {
		return errors.Wrap(err, "session configuration invalid")
	}

	// Grab the prompt server.
	promptServer := promptsvc.DefaultServer()

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := promptServer.RegisterPrompter(&createStreamPrompter{stream})
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
	promptServer.UnregisterPrompter(prompter)

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

// List lists existing sessions.
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

// Pause pauses existing sessions.
func (s *Server) Pause(_ context.Context, request *PauseRequest) (*PauseResponse, error) {
	// Perform pausing.
	// TODO: Figure out a way to monitor for cancellation.
	if err := s.manager.Pause(request.Specifications); err != nil {
		return nil, err
	}

	// Success.
	return &PauseResponse{}, nil
}

// Resume resumes existing sessions.
func (s *Server) Resume(stream Sessions_ResumeServer) error {
	// Receive the first request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Grab the prompt server.
	promptServer := promptsvc.DefaultServer()

	// Wrap the stream in a prompter and register it with the prompt server.
	prompter, err := promptServer.RegisterPrompter(&resumeStreamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Perform resuming.
	// TODO: Figure out a way to monitor for cancellation.
	err = s.manager.Resume(request.Specifications, prompter)

	// Unregister the prompter.
	promptServer.UnregisterPrompter(prompter)

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

// Terminate terminates existing sessions.
func (s *Server) Terminate(_ context.Context, request *TerminateRequest) (*TerminateResponse, error) {
	// Perform termination.
	// TODO: Figure out a way to monitor for cancellation.
	if err := s.manager.Terminate(request.Specifications); err != nil {
		return nil, err
	}

	// Success.
	return &TerminateResponse{}, nil
}
