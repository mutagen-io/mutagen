package session

import (
	"sort"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/ssh"
	"github.com/havoc-io/mutagen/state"
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

type createPrompter struct {
	stream *rpc.HandlerStream
}

func (p *createPrompter) Prompt(message, prompt string) (string, error) {
	// Create the request.
	request := CreateResponse{
		Challenge: &PromptRequest{
			Message: message,
			Prompt:  prompt,
		},
	}

	// Send the request.
	if err := p.stream.Encode(request); err != nil {
		return "", errors.Wrap(err, "unable to send prompt request")
	}

	// Receive the response.
	var response CreateRequest
	if err := p.stream.Decode(&response); err != nil {
		return "", errors.Wrap(err, "unable to receive prompt response")
	} else if response.Response == nil {
		return "", errors.New("invalid prompt response")
	}

	// Success.
	return response.Response.Response, nil
}

func (s *Service) create(stream *rpc.HandlerStream) {
	// Create an error transmitter.
	sendError := func(err error) {
		stream.Encode(CreateResponse{Error: err.Error()})
	}

	// Receive the request.
	var request CreateRequest
	if err := stream.Decode(&request); err != nil {
		sendError(errors.Wrap(err, "unable to receive request"))
		return
	}

	//  Wrap the stream in a prompter and register it with the SSH service.
	prompter := s.sshService.RegisterPrompter(&createPrompter{stream})

	// Attempt to create a session.
	controller, err := newSession(s.tracker, request.Alpha, request.Beta, request.Ignores, prompter)

	// Unregister the prompter.
	s.sshService.UnregisterPrompter(prompter)

	// Handle any creation error.
	if err != nil {
		sendError(err)
		return
	}

	// Register the controller.
	s.sessionsLock.Lock()
	s.sessions[controller.session.Identifier] = controller
	s.sessionsLock.Unlock()

	// Success.
	stream.Encode(CreateResponse{})
}

// byCreationTime implements the sort interface for SessionState, sorting
// sessions by creation time.
type byCreationTime []SessionState

func (s byCreationTime) Len() int {
	return len(s)
}

func (s byCreationTime) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byCreationTime) Less(i, j int) bool {
	// This comparison relies on the fact that Nanos can't be negative (at least
	// not according to the Protocol Buffers definition of its value). If Nanos
	// could be negative, we'd have to consider cases where seconds were equal
	// or within 1 of each other.
	return s[i].Session.CreationTime.Seconds < s[j].Session.CreationTime.Seconds ||
		(s[i].Session.CreationTime.Seconds == s[j].Session.CreationTime.Seconds &&
			s[i].Session.CreationTime.Nanos < s[j].Session.CreationTime.Nanos)
}

func (s *Service) list(stream *rpc.HandlerStream) {
	// Create an error transmitter.
	sendError := func(err error) {
		stream.Encode(ListResponse{Error: err.Error()})
	}

	// Receive the request.
	var request ListRequest
	if err := stream.Decode(&request); err != nil {
		sendError(errors.Wrap(err, "unable to receive request"))
		return
	}

	// Wait until the state has changed.
	stateIndex := s.tracker.WaitForChange(request.PreviousStateIndex)

	// Lock the session registry, grab session states, and then unlock the
	// registry. It's very important that we unlock without a notification here,
	// otherwise we'd trigger an infinite cycle of list/notify.
	s.sessionsLock.Lock()
	var sessions []SessionState
	for _, controller := range s.sessions {
		sessions = append(sessions, controller.currentState())
	}
	s.sessionsLock.UnlockWithoutNotify()

	// Sort sessions by creation time.
	sort.Sort(byCreationTime(sessions))

	// Done.
	stream.Encode(ListResponse{
		StateIndex: stateIndex,
		Sessions:   sessions,
	})
}

func (s *Service) pause(stream *rpc.HandlerStream) {
	// Create an error transmitter.
	sendError := func(err error) {
		stream.Encode(PauseResponse{Error: err.Error()})
	}

	// Receive the request.
	var request PauseRequest
	if err := stream.Decode(&request); err != nil {
		sendError(errors.Wrap(err, "unable to receive request"))
		return
	}

	// Lock the session registry and try to find the specified controller.
	s.sessionsLock.Lock()
	controller, ok := s.sessions[request.Session]
	s.sessionsLock.UnlockWithoutNotify()

	// If we couldn't find the controller, abort.
	if !ok {
		sendError(errors.New("unable to find session"))
		return
	}

	// Attempt to pause the session.
	if err := controller.halt(haltModePause); err != nil {
		sendError(errors.Wrap(err, "unable to pause session"))
		return
	}

	// Success.
	stream.Encode(PauseResponse{})
}

type resumePrompter struct {
	stream *rpc.HandlerStream
}

func (p *resumePrompter) Prompt(message, prompt string) (string, error) {
	// Create the request.
	request := ResumeResponse{
		Challenge: &PromptRequest{
			Message: message,
			Prompt:  prompt,
		},
	}

	// Send the request.
	if err := p.stream.Encode(request); err != nil {
		return "", errors.Wrap(err, "unable to send prompt request")
	}

	// Receive the response.
	var response ResumeRequest
	if err := p.stream.Decode(&response); err != nil {
		return "", errors.Wrap(err, "unable to receive prompt response")
	} else if response.Response == nil {
		return "", errors.New("invalid prompt response")
	}

	// Success.
	return response.Response.Response, nil
}

func (s *Service) resume(stream *rpc.HandlerStream) {
	// Create an error transmitter.
	sendError := func(err error) {
		stream.Encode(ResumeResponse{Error: err.Error()})
	}

	// Receive the request.
	var request ResumeRequest
	if err := stream.Decode(&request); err != nil {
		sendError(errors.Wrap(err, "unable to receive request"))
		return
	}

	// Lock the session registry and try to find the specified controller.
	s.sessionsLock.Lock()
	controller, ok := s.sessions[request.Session]
	s.sessionsLock.UnlockWithoutNotify()

	// If we couldn't find the controller, abort.
	if !ok {
		sendError(errors.New("unable to find session"))
		return
	}

	//  Wrap the stream in a prompter and register it with the SSH service.
	prompter := s.sshService.RegisterPrompter(&resumePrompter{stream})

	// Attempt to resume.
	err := controller.resume(prompter)

	// Unregister the prompter.
	s.sshService.UnregisterPrompter(prompter)

	// Handle any resume error.
	if err != nil {
		sendError(err)
		return
	}

	// Success.
	stream.Encode(ResumeResponse{})
}

func (s *Service) terminate(stream *rpc.HandlerStream) {
	// Create an error transmitter.
	sendError := func(err error) {
		stream.Encode(TerminateResponse{Error: err.Error()})
	}

	// Receive the request.
	var request TerminateRequest
	if err := stream.Decode(&request); err != nil {
		sendError(errors.Wrap(err, "unable to receive request"))
		return
	}

	// Lock the session registry and try to find the specified controller.
	s.sessionsLock.Lock()
	controller, ok := s.sessions[request.Session]
	s.sessionsLock.UnlockWithoutNotify()

	// If we couldn't find the controller, abort.
	if !ok {
		sendError(errors.New("unable to find session"))
		return
	}

	// Attempt to terminate the session.
	if err := controller.halt(haltModeTerminate); err != nil {
		sendError(errors.Wrap(err, "unable to terminate session"))
		return
	}

	// Since we termianted the session, we're responsible for unregistering it.
	s.sessionsLock.Lock()
	delete(s.sessions, request.Session)
	s.sessionsLock.Unlock()

	// Success.
	stream.Encode(TerminateResponse{})
}
