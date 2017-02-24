package session

import (
	"sort"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/ssh"
	"github.com/havoc-io/mutagen/state"
	"github.com/havoc-io/mutagen/timestamp"
)

const (
	MethodCreate    = "session.Create"
	MethodList      = "session.List"
	MethodPause     = "session.Pause"
	MethodResume    = "session.Resume"
	MethodTerminate = "session.Terminate"
)

type Service struct {
	// sshService performs registration and deregistration of prompters.
	sshService *ssh.Service
	// tracker tracks changes to session states.
	tracker *state.Tracker
	// sessionLock locks the sessions registry.
	sessionsLock *state.TrackingLock
	// sessions maps sessions to their respective controllers.
	sessions map[string]*controller
}

func NewService(sshService *ssh.Service) (*Service, error) {
	// Create a tracker and corresponding lock to watch for state changes.
	tracker := state.NewTracker()
	sessionsLock := state.NewTrackingLock(tracker)

	// Create the session registry.
	sessions := make(map[string]*controller)

	// Load existing sessions.
	sessionsDirectory, err := pathForSession("")
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute sessions directory")
	}
	identifiers, err := filesystem.DirectoryContents(sessionsDirectory)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read contents of sessions directory")
	}
	for _, identifier := range identifiers {
		if controller, err := loadSession(tracker, identifier); err != nil {
			continue
		} else {
			sessions[identifier] = controller
		}
	}

	// Success.
	return &Service{
		sshService:   sshService,
		tracker:      tracker,
		sessionsLock: sessionsLock,
		sessions:     sessions,
	}, nil
}

func (s *Service) Methods() map[string]rpc.Handler {
	return map[string]rpc.Handler{
		MethodCreate:    s.create,
		MethodList:      s.list,
		MethodPause:     s.pause,
		MethodResume:    s.resume,
		MethodTerminate: s.terminate,
	}
}

func (s *Service) Shutdown() {
	// Grab the registry lock and defer its release.
	s.sessionsLock.Lock()
	s.sessionsLock.UnlockWithoutNotify()

	// Attempt to halt each session so that it can shutdown cleanly. Ignore but
	// log any that fail to halt.
	for _, controller := range s.sessions {
		if err := controller.halt(haltModeShutdown); err != nil {
			// TODO: Log this halt failure.
		}
	}
}

type streamPrompter struct {
	stream rpc.HandlerStream
}

func (p *streamPrompter) Prompt(message, prompt string) (string, error) {
	// Create the request.
	request := PromptRequest{
		Message: message,
		Prompt:  prompt,
	}

	// Send the request.
	if err := p.stream.Send(request); err != nil {
		return "", errors.Wrap(err, "unable to send challenge")
	}

	// Receive the response.
	var response PromptResponse
	if err := p.stream.Receive(&response); err != nil {
		return "", errors.Wrap(err, "unable to receive response")
	}

	// Success.
	return response.Response, nil
}

func (s *Service) create(stream rpc.HandlerStream) error {
	// Receive the request.
	var request CreateRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	//  Wrap the stream in a prompter and register it with the SSH service.
	prompter := s.sshService.RegisterPrompter(&streamPrompter{stream})

	// Attempt to create a session.
	controller, err := newSession(
		s.tracker,
		request.Alpha, request.Beta,
		request.Ignores,
		prompter,
	)

	// Unregister the prompter.
	s.sshService.UnregisterPrompter(prompter)

	// Handle any creation error.
	if err != nil {
		return err
	}

	// Register the controller.
	s.sessionsLock.Lock()
	s.sessions[controller.session.Identifier] = controller
	s.sessionsLock.Unlock()

	// Success. We signal the end of the stream by closing it (which sends an
	// io.EOF), and returning from the handler will do that by default.
	return nil
}

func (s *Service) list(stream rpc.HandlerStream) error {
	// Receive the request.
	var request ListRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Loop indefinitely and track state changes. We'll bail after a single
	// response if monitoring wasn't requested.
	previousStateIndex := uint64(0)
	for {
		// Wait for a state change.
		// TODO: If the client disconnects while this handler is polling for
		// changes, this Goroutine will wait here until there's another change,
		// and will then exit when it tries (and fails) to send a response. This
		// will be fine in practice, but it's not elegant.
		previousStateIndex = s.tracker.WaitForChange(previousStateIndex)

		// Lock the session registry.
		s.sessionsLock.Lock()

		// Create a snapshot of the necessary session states.
		var sessions []SessionState
		var err error
		if request.Session != "" {
			if controller, ok := s.sessions[request.Session]; ok {
				sessions = append(sessions, controller.currentState())
			} else {
				err = errors.New("unable to find requested session")
			}
		} else {
			for _, controller := range s.sessions {
				sessions = append(sessions, controller.currentState())
			}
		}

		// Unlock the session registry. It's very important that we unlock
		// without a notification here, otherwise we'll trigger an infinite
		// cycle of state changes.
		s.sessionsLock.UnlockWithoutNotify()

		// Handle errors.
		if err != nil {
			return err
		}

		// Sort sessions by creation time.
		sort.Slice(sessions, func(i, j int) bool {
			return timestamp.Less(
				sessions[i].Session.CreationTime,
				sessions[j].Session.CreationTime,
			)
		})

		// Send this response.
		if err := stream.Send(ListResponse{Sessions: sessions}); err != nil {
			return errors.Wrap(err, "unable to send list response")
		}

		// If monitoring wasn't requested, then we're done.
		if !request.Monitor {
			return nil
		}

		// Otherwise wait for another (empty) request from the client as a
		// backpressure mechanism so we don't send more messages than it can
		// handle.
		var readyRequest ListRequest
		if err := stream.Receive(&readyRequest); err != nil {
			return errors.Wrap(err, "unable to receive ready request")
		}
	}
}

func (s *Service) pause(stream rpc.HandlerStream) error {
	// Receive the request.
	var request PauseRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Lock the session registry and try to find the specified controller.
	s.sessionsLock.Lock()
	controller, ok := s.sessions[request.Session]
	s.sessionsLock.UnlockWithoutNotify()

	// If we couldn't find the controller, abort.
	if !ok {
		return errors.New("unable to find session")
	}

	// Attempt to pause the session.
	if err := controller.halt(haltModePause); err != nil {
		return errors.Wrap(err, "unable to pause session")
	}

	// Success.
	return stream.Send(PauseResponse{})
}

func (s *Service) resume(stream rpc.HandlerStream) error {
	// Receive the request.
	var request ResumeRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Lock the session registry and try to find the specified controller.
	s.sessionsLock.Lock()
	controller, ok := s.sessions[request.Session]
	s.sessionsLock.UnlockWithoutNotify()

	// If we couldn't find the controller, abort.
	if !ok {
		return errors.New("unable to find session")
	}

	//  Wrap the stream in a prompter and register it with the SSH service.
	prompter := s.sshService.RegisterPrompter(&streamPrompter{stream})

	// Attempt to resume.
	err := controller.resume(prompter)

	// Unregister the prompter.
	s.sshService.UnregisterPrompter(prompter)

	// Handle any resume error.
	if err != nil {
		return err
	}

	// Success. We signal the end of the stream by closing it (which sends an
	// io.EOF), and returning from the handler will do that by default.
	return nil
}

func (s *Service) terminate(stream rpc.HandlerStream) error {
	// Receive the request.
	var request TerminateRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Lock the session registry and try to find the specified controller.
	s.sessionsLock.Lock()
	controller, ok := s.sessions[request.Session]
	s.sessionsLock.UnlockWithoutNotify()

	// If we couldn't find the controller, abort.
	if !ok {
		return errors.New("unable to find session")
	}

	// Attempt to terminate the session.
	if err := controller.halt(haltModeTerminate); err != nil {
		return errors.Wrap(err, "unable to terminate session")
	}

	// Since we termianted the session, we're responsible for unregistering it.
	s.sessionsLock.Lock()
	delete(s.sessions, request.Session)
	s.sessionsLock.Unlock()

	// Done.
	return stream.Send(TerminateResponse{})
}
