package session

import (
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/rpc"
	"github.com/havoc-io/mutagen/pkg/ssh"
	"github.com/havoc-io/mutagen/pkg/state"
)

const (
	MethodCreate    = "session.Create"
	MethodList      = "session.List"
	MethodPause     = "session.Pause"
	MethodResume    = "session.Resume"
	MethodTerminate = "session.Terminate"

	listThrottleInterval = 100 * time.Millisecond
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

func (s *Service) allControllers() []*controller {
	// Grab the registry lock and defer its release.
	s.sessionsLock.Lock()
	defer s.sessionsLock.UnlockWithoutNotify()

	// Generate a list of all controllers.
	controllers := make([]*controller, 0, len(s.sessions))
	for _, controller := range s.sessions {
		controllers = append(controllers, controller)
	}

	// Done.
	return controllers
}

const (
	minimumSessionMatchLength = 5
)

func fuzzyMatch(query string, controller *controller) (bool, bool, error) {
	// Don't allow empty or short strings to match anything.
	if query == "" {
		return false, false, errors.New("empty session specification is invalid")
	} else if len(query) < minimumSessionMatchLength {
		return false, false, errors.Errorf(
			"session specification must be at least %d characters",
			minimumSessionMatchLength,
		)
	}

	// Check for an exact match.
	exact := controller.session.Identifier == query

	// Check for a fuzzy match.
	fuzzy := strings.HasPrefix(controller.session.Identifier, query) ||
		strings.Contains(controller.session.Alpha.Path, query) ||
		strings.Contains(controller.session.Beta.Path, query) ||
		strings.Contains(controller.session.Alpha.Hostname, query) ||
		strings.Contains(controller.session.Beta.Hostname, query)

	// Done.
	return exact, fuzzy, nil
}

func (s *Service) findControllers(queries []string) ([]*controller, error) {
	// Grab the registry lock and defer its release.
	s.sessionsLock.Lock()
	defer s.sessionsLock.UnlockWithoutNotify()

	// Generate a list of controllers matching the specified queries.
	controllers := make([]*controller, 0, len(queries))
	for _, query := range queries {
		var match *controller
		for _, controller := range s.sessions {
			if exact, fuzzy, err := fuzzyMatch(query, controller); err != nil {
				return nil, err
			} else if exact {
				match = controller
				break
			} else if fuzzy {
				if match != nil {
					return nil, errors.Errorf("query \"%s\" matches multiple sessions", query)
				}
				match = controller
			}
		}
		if match == nil {
			return nil, errors.Errorf("query \"%s\" doesn't match any sessions", query)
		}
		controllers = append(controllers, match)
	}

	// Done.
	return controllers, nil
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
	// Poison state tracking to terminate monitoring.
	s.tracker.Poison()

	// Grab the registry lock and defer its release.
	s.sessionsLock.Lock()
	defer s.sessionsLock.UnlockWithoutNotify()

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
	// Receive and validate the request.
	var request CreateRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if err = request.Alpha.EnsureValid(); err != nil {
		return errors.Wrap(err, "unable to validate alpha URL")
	} else if err = request.Beta.EnsureValid(); err != nil {
		return errors.Wrap(err, "unable to validate beta URL")
	}

	//  Wrap the stream in a prompter and register it with the SSH service.
	prompter, err := s.sshService.RegisterPrompter(&streamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

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

	// Extract the session identifier.
	sessionId := controller.session.Identifier

	// Register the controller.
	s.sessionsLock.Lock()
	s.sessions[sessionId] = controller
	s.sessionsLock.Unlock()

	// Signal prompt completion.
	if err := stream.Send(PromptRequest{Done: true}); err != nil {
		return errors.Wrap(err, "unable to terminate prompting stream")
	}

	// Send the response.
	if err := stream.Send(CreateResponse{Session: sessionId}); err != nil {
		return errors.Wrap(err, "unable to send create response")
	}

	// Success.
	return nil
}

func (s *Service) list(stream rpc.HandlerStream) error {
	// Receive and validate the request.
	var request ListRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if request.All && len(request.SessionQueries) > 0 {
		return errors.New("all sessions requested with specifications provided")
	}

	// Wait for a state change from the previous index.
	stateIndex, poisoned := s.tracker.WaitForChange(request.PreviousStateIndex)
	if poisoned {
		return errors.New("state tracking terminated")
	}

	// Extract the controllers for the sessions of interest.
	var controllers []*controller
	if request.All {
		controllers = s.allControllers()
	} else if cs, err := s.findControllers(request.SessionQueries); err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	} else {
		controllers = cs
	}

	// Extract the state from each controller.
	states := make([]SessionState, 0, len(controllers))
	for _, controller := range controllers {
		states = append(states, controller.currentState())
	}

	// Sort session states by session creation time.
	sort.Slice(states, func(i, j int) bool {
		iTime := states[i].Session.CreationTime
		jTime := states[j].Session.CreationTime
		return iTime.Seconds < jTime.Seconds ||
			(iTime.Seconds == jTime.Seconds && iTime.Nanos < jTime.Nanos)
	})

	// Send the response.
	if err := stream.Send(ListResponse{StateIndex: stateIndex, SessionStates: states}); err != nil {
		return errors.Wrap(err, "unable to send list response")
	}

	// Success.
	return nil
}

func (s *Service) pause(stream rpc.HandlerStream) error {
	// Receive and validate the request.
	var request PauseRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if request.All && len(request.SessionQueries) > 0 {
		return errors.New("all sessions requested with specifications provided")
	}

	// Extract the controllers for the sessions of interest.
	var controllers []*controller
	if request.All {
		controllers = s.allControllers()
	} else if cs, err := s.findControllers(request.SessionQueries); err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	} else {
		controllers = cs
	}

	// Attempt to pause the sessions.
	for _, controller := range controllers {
		if err := controller.halt(haltModePause); err != nil {
			return errors.Wrap(err, "unable to pause session")
		}
	}

	// Send the response.
	if err := stream.Send(PauseResponse{}); err != nil {
		return errors.Wrap(err, "unable to send pause response")
	}

	// Success.
	return nil
}

func (s *Service) resume(stream rpc.HandlerStream) error {
	// Receive and validate the request.
	var request ResumeRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if request.All && len(request.SessionQueries) > 0 {
		return errors.New("all sessions requested with specifications provided")
	}

	// Extract the controllers for the sessions of interest.
	var controllers []*controller
	if request.All {
		controllers = s.allControllers()
	} else if cs, err := s.findControllers(request.SessionQueries); err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	} else {
		controllers = cs
	}

	//  Wrap the stream in a prompter and register it with the SSH service.
	prompter, err := s.sshService.RegisterPrompter(&streamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Attempt to resume.
	for _, controller := range controllers {
		if err := controller.resume(prompter); err != nil {
			s.sshService.UnregisterPrompter(prompter)
			return errors.Wrap(err, "unable to resume session")
		}
	}

	// Unregister the prompter.
	s.sshService.UnregisterPrompter(prompter)

	// Signal prompt completion.
	if err := stream.Send(PromptRequest{Done: true}); err != nil {
		return errors.Wrap(err, "unable to terminate prompting stream")
	}

	// Send the response.
	if err := stream.Send(ResumeResponse{}); err != nil {
		return errors.Wrap(err, "unable to send resume response")
	}

	// Success.
	return nil
}

func (s *Service) terminate(stream rpc.HandlerStream) error {
	// Receive and validate the request.
	var request TerminateRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	} else if request.All && len(request.SessionQueries) > 0 {
		return errors.New("all sessions requested with specifications provided")
	}

	// Extract the controllers for the sessions of interest.
	var controllers []*controller
	if request.All {
		controllers = s.allControllers()
	} else if cs, err := s.findControllers(request.SessionQueries); err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	} else {
		controllers = cs
	}

	// Attempt to terminate the sessions. Since we're terminating them, we're
	// responsible for removing them from the session map.
	for _, controller := range controllers {
		if err := controller.halt(haltModeTerminate); err != nil {
			return errors.Wrap(err, "unable to terminate session")
		}
		s.sessionsLock.Lock()
		delete(s.sessions, controller.session.Identifier)
		s.sessionsLock.Unlock()
	}

	// Send the response.
	if err := stream.Send(TerminateResponse{}); err != nil {
		return errors.Wrap(err, "unable to send terminate response")
	}

	// Success.
	return nil
}
