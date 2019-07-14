package forwarding

import (
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/google/uuid"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/logging"
	"github.com/havoc-io/mutagen/pkg/selection"
	"github.com/havoc-io/mutagen/pkg/state"
	"github.com/havoc-io/mutagen/pkg/url"
)

// Manager provides forwarding session management facilities. Its methods are
// safe for concurrent usage, so it can be easily exported via an RPC interface.
type Manager struct {
	// logger is the underlying logger.
	logger *logging.Logger
	// tracker tracks changes to session states.
	tracker *state.Tracker
	// sessionLock locks the sessions registry.
	sessionsLock *state.TrackingLock
	// sessions maps sessions to their respective controllers.
	sessions map[string]*controller
}

// NewManager creates a new Manager instance.
func NewManager(logger *logging.Logger) (*Manager, error) {
	// Create a tracker and corresponding lock to watch for state changes.
	tracker := state.NewTracker()
	sessionsLock := state.NewTrackingLock(tracker)

	// Create the session registry.
	sessions := make(map[string]*controller)

	// Load existing sessions.
	logger.Println("Looking for existing sessions")
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
		logger.Println("Loading session", identifier)
		if controller, err := loadSession(logger.Sublogger(identifier), tracker, identifier); err != nil {
			continue
		} else {
			sessions[identifier] = controller
		}
	}

	// Success.
	logger.Println("Session manager initialized")
	return &Manager{
		logger:       logger,
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
				strings.Contains(controller.session.Source.Path, specification) ||
				strings.Contains(controller.session.Destination.Path, specification) ||
				strings.Contains(controller.session.Source.Host, specification) ||
				strings.Contains(controller.session.Destination.Host, specification)
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
func (m *Manager) findControllersByLabelSelector(labelSelector string) ([]*controller, error) {
	// Parse the label selector.
	selector, err := selection.ParseLabelSelector(labelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse label selector")
	}

	// Grab the registry lock and defer its release.
	m.sessionsLock.Lock()
	defer m.sessionsLock.UnlockWithoutNotify()

	// Loop over controllers and look for matches.
	var controllers []*controller
	for _, controller := range m.sessions {
		if selector.Matches(controller.session.Labels) {
			controllers = append(controllers, controller)
		}
	}

	// Done.
	return controllers, nil
}

// selectControllers generates a list of controllers using the mechanism
// specified by the provided selection.
func (m *Manager) selectControllers(selection *selection.Selection) ([]*controller, error) {
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
	// Log the shutdown.
	m.logger.Println("Shutting down")

	// Poison state tracking to terminate monitoring.
	m.tracker.Poison()

	// Grab the registry lock and defer its release.
	m.sessionsLock.Lock()
	defer m.sessionsLock.UnlockWithoutNotify()

	// Attempt to halt each session so that it can shutdown cleanly. Ignore but
	// log any that fail to halt.
	for _, controller := range m.sessions {
		m.logger.Println("Halting session", controller.session.Identifier)
		if err := controller.halt(controllerHaltModeShutdown, ""); err != nil {
			// TODO: Log this halt failure.
		}
	}
}

// Create tells the manager to create a new session.
func (m *Manager) Create(
	source, destination *url.URL,
	configuration, configurationSource, configurationDestination *Configuration,
	labels map[string]string,
	prompter string,
) (string, error) {
	// Create a unique session identifier.
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "unable to generate UUID for session")
	}
	identifier := randomUUID.String()

	// Attempt to create a session.
	controller, err := newSession(
		m.logger.Sublogger(identifier),
		m.tracker,
		identifier,
		source, destination,
		configuration, configurationSource, configurationDestination,
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
func (m *Manager) List(selection *selection.Selection, previousStateIndex uint64) (uint64, []*State, error) {
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

// Pause tells the manager to pause sessions matching the given specifications.
func (m *Manager) Pause(selection *selection.Selection, prompter string) error {
	// Extract the controllers for the sessions of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	}

	// Attempt to pause the sessions.
	for _, controller := range controllers {
		if err := controller.halt(controllerHaltModePause, prompter); err != nil {
			return errors.Wrap(err, "unable to pause session")
		}
	}

	// Success.
	return nil
}

// Resume tells the manager to resume sessions matching the given
// specifications.
func (m *Manager) Resume(selection *selection.Selection, prompter string) error {
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
func (m *Manager) Terminate(selection *selection.Selection, prompter string) error {
	// Extract the controllers for the sessions of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	}

	// Attempt to terminate the sessions. Since we're terminating them, we're
	// responsible for removing them from the session map.
	for _, controller := range controllers {
		if err := controller.halt(controllerHaltModeTerminate, prompter); err != nil {
			return errors.Wrap(err, "unable to terminate session")
		}
		m.sessionsLock.Lock()
		delete(m.sessions, controller.session.Identifier)
		m.sessionsLock.Unlock()
	}

	// Success.
	return nil
}
