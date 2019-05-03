package session

import (
	contextpkg "context"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/labels"

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

// NewManager creates a new manager instance.
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
	sessionsDirectoryContents, err := filesystem.DirectoryContentsByPath(sessionsDirectory)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read contents of sessions directory")
	}
	for _, c := range sessionsDirectoryContents {
		// TODO: Ensure that the name matches the expected format.
		identifier := c.Name()
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

// allControllers creates a list of all controllers managed by the manager.
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
	// minimumSessionSpecificationLength is the minimum session specification
	// length needed for a fuzzy match.
	minimumSessionSpecificationLength = 5
)

// findControllersBySpecification generates a list of controllers matching the
// given specifications.
func (m *Manager) findControllersBySpecification(specifications []string) ([]*controller, error) {
	// Grab the registry lock and defer its release.
	m.sessionsLock.Lock()
	defer m.sessionsLock.UnlockWithoutNotify()

	// Generate a list of controllers matching the specified specifications.
	controllers := make([]*controller, 0, len(specifications))
	for _, specification := range specifications {
		// Validate the specification.
		if specification == "" {
			return nil, errors.New("empty session specification is invalid")
		} else if len(specification) < minimumSessionSpecificationLength {
			return nil, errors.Errorf(
				"session specification must be at least %d characters",
				minimumSessionSpecificationLength,
			)
		}

		// Attempt to find a match.
		var match *controller
		for _, controller := range m.sessions {
			// Check for an exact match.
			if controller.session.Identifier == specification {
				match = controller
				break
			}

			// Check for a fuzzy match.
			fuzzy := strings.HasPrefix(controller.session.Identifier, specification) ||
				strings.Contains(controller.session.Alpha.Path, specification) ||
				strings.Contains(controller.session.Beta.Path, specification) ||
				strings.Contains(controller.session.Alpha.Hostname, specification) ||
				strings.Contains(controller.session.Beta.Hostname, specification)
			if fuzzy {
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

// findControllersByLabelSelector generates a list of controllers using the
// specified label selector.
func (m *Manager) findControllersByLabelSelector(rawSelector string) ([]*controller, error) {
	// Parse the label selector.
	selector, err := labels.Parse(rawSelector)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse label selector")
	}

	// Grab the registry lock and defer its release.
	m.sessionsLock.Lock()
	defer m.sessionsLock.UnlockWithoutNotify()

	// Loop over controllers and look for matches.
	var controllers []*controller
	for _, controller := range m.sessions {
		if selector.Matches(labels.Set(controller.session.Labels)) {
			controllers = append(controllers, controller)
		}
	}

	// Done.
	return controllers, nil
}

// selectControllers generates a list of controllers using the mechanism
// specified by the provided selection.
func (m *Manager) selectControllers(selection *Selection) ([]*controller, error) {
	// Dispatch selection based on the requested mechanism.
	if selection.All {
		return m.allControllers(), nil
	} else if len(selection.Specifications) > 0 {
		return m.findControllersBySpecification(selection.Specifications)
	} else if selection.LabelSelector != "" {
		return m.findControllersByLabelSelector(selection.LabelSelector)
	} else {
		// TODO: Should we panic here instead?
		return nil, errors.New("invalid session selection")
	}
}

// Shutdown tells the manager to gracefully halt sessions.
func (m *Manager) Shutdown() {
	// Poison state tracking to terminate monitoring.
	m.tracker.Poison()

	// Grab the registry lock and defer its release.
	m.sessionsLock.Lock()
	defer m.sessionsLock.UnlockWithoutNotify()

	// Attempt to halt each session so that it can shutdown cleanly. Ignore but
	// log any that fail to halt.
	for _, controller := range m.sessions {
		if err := controller.halt(haltModeShutdown, ""); err != nil {
			// TODO: Log this halt failure.
		}
	}
}

// Create tells the manager to create a new session.
func (m *Manager) Create(
	alpha, beta *url.URL,
	configuration, configurationAlpha, configurationBeta *Configuration,
	labels map[string]string,
	prompter string,
) (string, error) {
	// Attempt to create a session.
	controller, err := newSession(
		m.tracker,
		alpha, beta,
		configuration, configurationAlpha, configurationBeta,
		labels,
		prompter,
	)
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

// List requests a state snapshot for the specified sessions.
func (m *Manager) List(selection *Selection, previousStateIndex uint64) (uint64, []*State, error) {
	// Wait for a state change from the previous index.
	stateIndex, poisoned := m.tracker.WaitForChange(previousStateIndex)
	if poisoned {
		return 0, nil, errors.New("state tracking terminated")
	}

	// Extract the controllers for the sessions of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return 0, nil, errors.Wrap(err, "unable to locate requested sessions")
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

// Flush tells the manager to flush sessions matching the given specifications.
func (m *Manager) Flush(selection *Selection, prompter string, skipWait bool, context contextpkg.Context) error {
	// Extract the controllers for the sessions of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	}

	// Attempt to flush the sessions.
	for _, controller := range controllers {
		if err := controller.flush(prompter, skipWait, context); err != nil {
			return errors.Wrap(err, "unable to flush session")
		}
	}

	// Success.
	return nil
}

// Pause tells the manager to pause sessions matching the given specifications.
func (m *Manager) Pause(selection *Selection, prompter string) error {
	// Extract the controllers for the sessions of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	}

	// Attempt to pause the sessions.
	for _, controller := range controllers {
		if err := controller.halt(haltModePause, prompter); err != nil {
			return errors.Wrap(err, "unable to pause session")
		}
	}

	// Success.
	return nil
}

// Resume tells the manager to resume sessions matching the given
// specifications.
func (m *Manager) Resume(selection *Selection, prompter string) error {
	// Extract the controllers for the sessions of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
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

// Terminate tells the manager to terminate sessions matching the given
// specifications.
func (m *Manager) Terminate(selection *Selection, prompter string) error {
	// Extract the controllers for the sessions of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	}

	// Attempt to terminate the sessions. Since we're terminating them, we're
	// responsible for removing them from the session map.
	for _, controller := range controllers {
		if err := controller.halt(haltModeTerminate, prompter); err != nil {
			return errors.Wrap(err, "unable to terminate session")
		}
		m.sessionsLock.Lock()
		delete(m.sessions, controller.session.Identifier)
		m.sessionsLock.Unlock()
	}

	// Success.
	return nil
}
