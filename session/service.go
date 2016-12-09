package session

import (
	"sort"

	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"github.com/havoc-io/mutagen/filesystem"
)

type Service struct {
	// stateTracker locks and tracks state changes made to the session map and
	// the individual sessions it contains.
	stateTracker *stateTracker
	// sessions maps session identifiers to their controllers.
	sessions map[string]*controller
}

func NewService() (*Service, error) {
	// Create the state tracker.
	stateTracker := newStateTracker()

	// Create the sessions map.
	sessions := make(map[string]*controller)

	// Load existing session identifiers.
	// HACK: We use some internal knowledge of pathForSession here (namely that
	// an empty identifier returns the session directory itself and the
	// semantics of its error values), and we assume anything in that directory
	// is a session.
	// TODO: Should we integrate this logic with controller? I think I'd rather
	// have the controller in charge of path logic honestly.
	sessionsDirectory, err := pathForSession("")
	if err != nil {
		return nil, err
	}
	identifiers, err := filesystem.DirectoryContents(sessionsDirectory)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read sessions directory")
	}

	// Load and register sessions. We don't need to hold the lock at this point
	// since nobody can access the registry yet. If a session fails to load,
	// then just ignore it.
	for _, identifier := range identifiers {
		if controller, err := loadSession(stateTracker, identifier); err != nil {
			continue
		} else {
			sessions[identifier] = controller
		}
	}

	// Success.
	return &Service{
		stateTracker: stateTracker,
		sessions:     sessions,
	}, nil
}

func (s *Service) shutdown() error {
	// TODO: Implement.
	return errors.New("not implemented")
}

func (s *Service) Start(context context.Context, request *StartRequest) (*StartResponse, error) {
	// Validate URLs. All we really need to do at this point is ensure their
	// paths are non-empty, because that's the only thing the session package
	// really knows enough to validate. Any other parameters can be validated at
	// dial-time.
	if request.Alpha.Path == "" {
		return nil, errors.New("alpha URL has empty path")
	} else if request.Beta.Path == "" {
		return nil, errors.New("beta URL has empty path")
	}

	// Attempt to create the session.
	controller, err := newSession(
		s.stateTracker,
		context,
		request.Alpha, request.Beta,
		request.Prompter,
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create session")
	}

	// Register the session.
	s.stateTracker.lock()
	s.sessions[controller.session.Identifier] = controller
	s.stateTracker.notifyOfChangesAndUnlock()

	// Success.
	return &StartResponse{}, nil
}

// byCreationDate implements the sort interface for SessionState, sorting
// sessions by creation date. It is used by the List handler.
type byCreationDate []*SessionState

func (d byCreationDate) Len() int {
	return len(d)
}

func (d byCreationDate) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d byCreationDate) Less(i, j int) bool {
	return d[i].Session.CreationTime.Before(*d[j].Session.CreationTime)
}

func (s *Service) List(_ context.Context, request *ListRequest) (*ListResponse, error) {
	// Wait until there is a change from the previous state and lock the state.
	newStateIndex := s.stateTracker.waitForChangeAndLock(request.PreviousStateIndex)

	// Create the initial response.
	response := &ListResponse{
		StateIndex: newStateIndex,
	}

	// Iterate through the session map and record the public state components.
	for _, controller := range s.sessions {
		response.Sessions = append(response.Sessions, controller.state())
	}

	// Unlock the state.
	s.stateTracker.unlock()

	// Sort the sessions by creation date.
	sort.Sort(byCreationDate(response.Sessions))

	// Success.
	return response, nil
}

func (s *Service) Pause(_ context.Context, request *PauseRequest) (*PauseResponse, error) {
	// Grab the relevant controller.
	s.stateTracker.lock()
	controller := s.sessions[request.Session]
	s.stateTracker.unlock()

	// Ensure that the controller is valid.
	if controller == nil {
		return nil, errors.New("session not found")
	}

	// Attempt to pause.
	err := controller.pause()
	if err != nil {
		return nil, err
	}

	// Success.
	return &PauseResponse{}, nil
}

func (s *Service) Resume(context context.Context, request *ResumeRequest) (*ResumeResponse, error) {
	// Grab the relevant controller.
	s.stateTracker.lock()
	controller := s.sessions[request.Session]
	s.stateTracker.unlock()

	// Ensure that the controller is valid.
	if controller == nil {
		return nil, errors.New("session not found")
	}

	// Attempt to resume.
	err := controller.resume(context, request.Prompter)
	if err != nil {
		return nil, err
	}

	// Success.
	return &ResumeResponse{}, nil
}

func (s *Service) Stop(_ context.Context, request *StopRequest) (*StopResponse, error) {
	// Grab the relevant controller.
	s.stateTracker.lock()
	controller := s.sessions[request.Session]
	s.stateTracker.unlock()

	// Ensure that the controller is valid.
	if controller == nil {
		return nil, errors.New("session not found")
	}

	// Attempt to stop.
	err := controller.stop(true)
	if err != nil {
		return nil, err
	}

	// Deregister the session.
	s.stateTracker.lock()
	delete(s.sessions, request.Session)
	s.stateTracker.notifyOfChangesAndUnlock()

	// Success.
	return &StopResponse{}, nil
}
