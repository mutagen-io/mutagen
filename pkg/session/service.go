package session

import (
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes"

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

const (
	minimumSessionMatchLength = 5
)

// findSession provides fuzzy session matching by converting a query string into
// a full session id. The query string is tried as a full session id, session id
// prefix, URL substring, and then host substring. If the specified query
// matches more than one session, it is considered ambiguous and an error is
// returned.
func (s *Service) findSession(query string) (string, error) {
	// Don't allow empty or short strings to match anything.
	if query == "" {
		return "", errors.New("empty session specification is invalid")
	} else if len(query) < minimumSessionMatchLength {
		return "", errors.Errorf(
			"session specification must be at least %d characters",
			minimumSessionMatchLength,
		)
	}

	// Grab the registry lock and defer its release.
	s.sessionsLock.Lock()
	defer s.sessionsLock.UnlockWithoutNotify()

	// Track a match.
	result := ""

	// Search for matches.
	for identifier, controller := range s.sessions {
		match := identifier == query ||
			strings.HasPrefix(identifier, query) ||
			strings.Contains(controller.session.Alpha.Path, query) ||
			strings.Contains(controller.session.Beta.Path, query) ||
			strings.Contains(controller.session.Alpha.Hostname, query) ||
			strings.Contains(controller.session.Beta.Hostname, query)
		if match {
			if result != "" {
				return "", errors.Errorf("query \"%s\" matches multiple sessions", query)
			}
			result = identifier
		}
	}

	// Success.
	return result, nil
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
	// Receive the request.
	var request CreateRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
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
	// Receive the request.
	var request ListRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Determine the session query of interest and whether or not this is a
	// repeated request. An empty session query after this point indicates that
	// all sessions are of interest.
	var sessionQuery string
	var repeated bool
	switch request.Kind {
	case ListRequestKindSingle:
		sessionQuery = request.SessionQuery
	case ListRequestKindRepeated:
		sessionQuery = request.SessionQuery
		repeated = true
	case ListRequestKindRepeatedLatest:
		var mostRecentSessionCreationTime time.Time
		s.sessionsLock.Lock()
		for _, controller := range s.sessions {
			state := controller.currentState()
			creationTime, err := ptypes.Timestamp(state.Session.CreationTime)
			if err != nil {
				s.sessionsLock.Unlock()
				return errors.Wrap(err, "unable to convert creation time format")
			}
			if creationTime.After(mostRecentSessionCreationTime) {
				sessionQuery = state.Session.Identifier
				mostRecentSessionCreationTime = creationTime
			}
		}
		s.sessionsLock.Unlock()
		if sessionQuery == "" {
			return errors.New("no sessions present")
		}
		repeated = true
	default:
		return errors.New("unknown list request kind")
	}

	// If a session query has been provided, resolve the full session identifier
	// to which it refers.
	session := ""
	if sessionQuery != "" {
		if s, err := s.findSession(sessionQuery); err != nil {
			return errors.Wrap(err, "unable to identify session")
		} else {
			session = s
		}
	}

	// Loop indefinitely and track state changes. We'll bail after a single
	// response if monitoring wasn't requested.
	previousStateIndex := uint64(0)
	var poisoned bool
	for {
		// Wait for a state change.
		// TODO: If the client disconnects while this handler is polling for
		// changes, this Goroutine will wait here until there's another change,
		// and will then exit when it tries (and fails) to send a response. This
		// will be fine in practice, but it's not elegant.
		previousStateIndex, poisoned = s.tracker.WaitForChange(previousStateIndex)
		if poisoned {
			return errors.New("state tracking terminated")
		}

		// Lock the session registry.
		s.sessionsLock.Lock()

		// Create a snapshot of the necessary session states.
		var sessions []SessionState
		var err error
		if session != "" {
			if controller, ok := s.sessions[session]; ok {
				sessions = append(sessions, controller.currentState())
			} else {
				err = errors.New("requested session no longer exists")
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
			iTime := sessions[i].Session.CreationTime
			jTime := sessions[j].Session.CreationTime
			return iTime.Seconds < jTime.Seconds ||
				(iTime.Seconds == jTime.Seconds && iTime.Nanos < jTime.Nanos)
		})

		// Send this response.
		if err := stream.Send(ListResponse{Sessions: sessions}); err != nil {
			return errors.Wrap(err, "unable to send list response")
		}

		// If repeated listings weren't requested, then we're done.
		if !repeated {
			return nil
		}

		// Perform a sleep to throttle list requests and wait for another
		// (empty) request from the client as a backpressure mechanism. Both of
		// these operations are necessary. The sleep protects the daemon and the
		// backpressure protects the client. In reality the sleep is probably
		// sufficient to protect the client, but you need a backpressure
		// mechanism to be sure.
		time.Sleep(listThrottleInterval)
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

	// Resolve the session query to a full session identifier.
	session, err := s.findSession(request.SessionQuery)
	if err != nil {
		return errors.Wrap(err, "unable to identify session")
	}

	// Lock the session registry and try to find the specified controller.
	s.sessionsLock.Lock()
	controller, ok := s.sessions[session]
	s.sessionsLock.UnlockWithoutNotify()

	// If we couldn't find the controller, abort.
	if !ok {
		return errors.New("unable to find session")
	}

	// Attempt to pause the session.
	if err := controller.halt(haltModePause); err != nil {
		return errors.Wrap(err, "unable to pause session")
	}

	// Send the response.
	if err := stream.Send(PauseResponse{}); err != nil {
		return errors.Wrap(err, "unable to send pause response")
	}

	// Success.
	return nil
}

func (s *Service) resume(stream rpc.HandlerStream) error {
	// Receive the request.
	var request ResumeRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Resolve the session query to a full session identifier.
	session, err := s.findSession(request.SessionQuery)
	if err != nil {
		return errors.Wrap(err, "unable to identify session")
	}

	// Lock the session registry and try to find the specified controller.
	s.sessionsLock.Lock()
	controller, ok := s.sessions[session]
	s.sessionsLock.UnlockWithoutNotify()

	// If we couldn't find the controller, abort.
	if !ok {
		return errors.New("unable to find session")
	}

	//  Wrap the stream in a prompter and register it with the SSH service.
	prompter, err := s.sshService.RegisterPrompter(&streamPrompter{stream})
	if err != nil {
		return errors.Wrap(err, "unable to register prompter")
	}

	// Attempt to resume.
	err = controller.resume(prompter)

	// Unregister the prompter.
	s.sshService.UnregisterPrompter(prompter)

	// Handle any resume error.
	if err != nil {
		return err
	}

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
	// Receive the request.
	var request TerminateRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Resolve the session query to a full session identifier.
	session, err := s.findSession(request.SessionQuery)
	if err != nil {
		return errors.Wrap(err, "unable to identify session")
	}

	// Lock the session registry and try to find the specified controller.
	s.sessionsLock.Lock()
	controller, ok := s.sessions[session]
	s.sessionsLock.UnlockWithoutNotify()

	// If we couldn't find the controller, abort.
	if !ok {
		return errors.New("unable to find session")
	}

	// Attempt to terminate the session.
	if err := controller.halt(haltModeTerminate); err != nil {
		return errors.Wrap(err, "unable to terminate session")
	}

	// Since we terminated the session, we're responsible for unregistering it.
	s.sessionsLock.Lock()
	delete(s.sessions, session)
	s.sessionsLock.Unlock()

	// Send the response.
	if err := stream.Send(TerminateResponse{}); err != nil {
		return errors.Wrap(err, "unable to send terminate response")
	}

	// Success.
	return nil
}
