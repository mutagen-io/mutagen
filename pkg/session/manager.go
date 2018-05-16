package session

import (
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/state"
	"github.com/havoc-io/mutagen/pkg/url"
)

// Manager provides session management facilities for the daemon. Its methods
// are safe for concurrent usage, so it can be easily exported via an RPC
// interface.
type Manager struct {
	// tracker tracks changes to session states.
	tracker *state.Tracker
	// sessionLock locks the sessions registry.
	sessionsLock *state.TrackingLock
	// sessions maps sessions to their respective controllers.
	sessions map[string]*controller
}

func NewManager() (*Manager, error) {
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
	return &Manager{
		tracker:      tracker,
		sessionsLock: sessionsLock,
		sessions:     sessions,
	}, nil
}

func (m *Manager) allControllers() []*controller {
	// Grab the registry lock and defer its release.
	m.sessionsLock.Lock()
	defer m.sessionsLock.UnlockWithoutNotify()

	// Generate a list of all controllers.
	controllers := make([]*controller, 0, len(m.sessions))
	for _, controller := range m.sessions {
		controllers = append(controllers, controller)
	}

	// Done.
	return controllers
}

const (
	minimumSessionSpecificationLength = 5
)

func fuzzyMatch(specification string, controller *controller) (bool, bool, error) {
	// Don't allow empty or short strings to match anything.
	if specification == "" {
		return false, false, errors.New("empty session specification is invalid")
	} else if len(specification) < minimumSessionSpecificationLength {
		return false, false, errors.Errorf(
			"session specification must be at least %d characters",
			minimumSessionSpecificationLength,
		)
	}

	// Check for an exact match.
	exact := controller.session.Identifier == specification

	// Check for a fuzzy match.
	fuzzy := strings.HasPrefix(controller.session.Identifier, specification) ||
		strings.Contains(controller.session.Alpha.Path, specification) ||
		strings.Contains(controller.session.Beta.Path, specification) ||
		strings.Contains(controller.session.Alpha.Hostname, specification) ||
		strings.Contains(controller.session.Beta.Hostname, specification)

	// Done.
	return exact, fuzzy, nil
}

func (m *Manager) findControllers(specifications []string) ([]*controller, error) {
	// Grab the registry lock and defer its release.
	m.sessionsLock.Lock()
	defer m.sessionsLock.UnlockWithoutNotify()

	// Generate a list of controllers matching the specified specifications.
	controllers := make([]*controller, 0, len(specifications))
	for _, specification := range specifications {
		var match *controller
		for _, controller := range m.sessions {
			if exact, fuzzy, err := fuzzyMatch(specification, controller); err != nil {
				return nil, err
			} else if exact {
				match = controller
				break
			} else if fuzzy {
				if match != nil {
					return nil, errors.Errorf("specification \"%s\" matches multiple sessions", specification)
				}
				match = controller
			}
		}
		if match == nil {
			return nil, errors.Errorf("specification \"%s\" doesn't match any sessions", specification)
		}
		controllers = append(controllers, match)
	}

	// Done.
	return controllers, nil
}

func (m *Manager) Shutdown() {
	// Poison state tracking to terminate monitoring.
	m.tracker.Poison()

	// Grab the registry lock and defer its release.
	m.sessionsLock.Lock()
	defer m.sessionsLock.UnlockWithoutNotify()

	// Attempt to halt each session so that it can shutdown cleanly. Ignore but
	// log any that fail to halt.
	for _, controller := range m.sessions {
		if err := controller.halt(haltModeShutdown); err != nil {
			// TODO: Log this halt failure.
		}
	}
}

func (m *Manager) Create(alpha, beta *url.URL, ignores []string, prompter string) (string, error) {
	// Attempt to create a session.
	controller, err := newSession(m.tracker, alpha, beta, ignores, prompter)
	if err != nil {
		return "", err
	}

	// Register the controller.
	m.sessionsLock.Lock()
	m.sessions[controller.session.Identifier] = controller
	m.sessionsLock.Unlock()

	// Done.
	return controller.session.Identifier, nil
}

func (m *Manager) List(previousStateIndex uint64, specifications []string) (uint64, []*State, error) {
	// Wait for a state change from the previous index.
	stateIndex, poisoned := m.tracker.WaitForChange(previousStateIndex)
	if poisoned {
		return 0, nil, errors.New("state tracking terminated")
	}

	// Extract the controllers for the sessions of interest.
	var controllers []*controller
	if len(specifications) == 0 {
		controllers = m.allControllers()
	} else if cs, err := m.findControllers(specifications); err != nil {
		return 0, nil, errors.Wrap(err, "unable to locate requested sessions")
	} else {
		controllers = cs
	}

	// Extract the state from each controller.
	states := make([]*State, len(controllers))
	for i, controller := range controllers {
		states[i] = controller.currentState()
	}

	// Sort session states by session creation time.
	sort.Slice(states, func(i, j int) bool {
		iTime := states[i].Session.CreationTime
		jTime := states[j].Session.CreationTime
		return iTime.Seconds < jTime.Seconds ||
			(iTime.Seconds == jTime.Seconds && iTime.Nanos < jTime.Nanos)
	})

	// Success.
	return stateIndex, states, nil
}

func (m *Manager) Pause(specifications []string) error {
	// Extract the controllers for the sessions of interest.
	var controllers []*controller
	if len(specifications) == 0 {
		controllers = m.allControllers()
	} else if cs, err := m.findControllers(specifications); err != nil {
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

	// Success.
	return nil
}

func (m *Manager) Resume(specifications []string, prompter string) error {
	// Extract the controllers for the sessions of interest.
	var controllers []*controller
	if len(specifications) == 0 {
		controllers = m.allControllers()
	} else if cs, err := m.findControllers(specifications); err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	} else {
		controllers = cs
	}

	// Attempt to resume.
	for _, controller := range controllers {
		if err := controller.resume(prompter); err != nil {
			return errors.Wrap(err, "unable to resume session")
		}
	}

	// Success.
	return nil
}

func (m *Manager) Terminate(specifications []string) error {
	// Extract the controllers for the sessions of interest.
	var controllers []*controller
	if len(specifications) == 0 {
		controllers = m.allControllers()
	} else if cs, err := m.findControllers(specifications); err != nil {
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
		m.sessionsLock.Lock()
		delete(m.sessions, controller.session.Identifier)
		m.sessionsLock.Unlock()
	}

	// Success.
	return nil
}
