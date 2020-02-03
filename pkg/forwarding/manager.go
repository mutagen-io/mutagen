package forwarding

import (
	"context"
	"sort"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/identifier"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/selection"
	"github.com/mutagen-io/mutagen/pkg/state"
	"github.com/mutagen-io/mutagen/pkg/url"
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
	logger.Info("Looking for existing sessions")
	sessionsDirectory, err := pathForSession("")
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute sessions directory")
	}
	sessionsDirectoryContents, err := filesystem.DirectoryContentsByPath(sessionsDirectory)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read contents of sessions directory")
	}
	for _, c := range sessionsDirectoryContents {
		identifier := c.Name()
		logger.Info("Loading session", identifier)
		if controller, err := loadSession(logger.Sublogger(identifier), tracker, identifier); err != nil {
			continue
		} else {
			sessions[identifier] = controller
		}
	}

	// Success.
	logger.Info("Session manager initialized")
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

// findControllersBySpecification generates a list of controllers matching the
// given specifications.
func (m *Manager) findControllersBySpecification(specifications []string) ([]*controller, error) {
	// Grab the registry lock and defer its release.
	m.sessionsLock.Lock()
	defer m.sessionsLock.UnlockWithoutNotify()

	// Generate a list of controllers matching the specifications. We allow each
	// specification to match multiple controllers, so we store matches in a set
	// before converting them to a list. We do require that each specification
	// match at least one controller.
	controllerSet := make(map[*controller]bool)
	for _, specification := range specifications {
		var matched bool
		for _, controller := range m.sessions {
			if controller.session.Identifier == specification || controller.session.Name == specification {
				controllerSet[controller] = true
				matched = true
			}
		}
		if !matched {
			return nil, errors.Errorf("specification \"%s\" did not match any sessions", specification)
		}
	}

	// Convert the set to a list.
	controllers := make([]*controller, 0, len(controllerSet))
	for c := range controllerSet {
		controllers = append(controllers, c)
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
	m.logger.Info("Shutting down")

	// Poison state tracking to terminate monitoring.
	m.tracker.Poison()

	// Grab the registry lock and defer its release.
	m.sessionsLock.Lock()
	defer m.sessionsLock.UnlockWithoutNotify()

	// Attempt to halt each session so that it can shutdown cleanly. Ignore but
	// log any that fail to halt.
	for _, controller := range m.sessions {
		m.logger.Info("Halting session", controller.session.Identifier)
		if err := controller.halt(context.Background(), controllerHaltModeShutdown, ""); err != nil {
			// TODO: Log this halt failure.
		}
	}
}

// Create tells the manager to create a new session.
func (m *Manager) Create(
	ctx context.Context,
	source, destination *url.URL,
	configuration, configurationSource, configurationDestination *Configuration,
	name string,
	labels map[string]string,
	paused bool,
	prompter string,
) (string, error) {
	// Create a unique session identifier.
	identifier, err := identifier.New(identifier.PrefixForwarding)
	if err != nil {
		return "", errors.Wrap(err, "unable to generate UUID for session")
	}

	// Attempt to create a session.
	controller, err := newSession(
		ctx,
		m.logger.Sublogger(identifier),
		m.tracker,
		identifier,
		source, destination,
		configuration, configurationSource, configurationDestination,
		name,
		labels,
		paused,
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
func (m *Manager) List(_ context.Context, selection *selection.Selection, previousStateIndex uint64) (uint64, []*State, error) {
	// Wait for a state change from the previous index.
	// TODO: Figure out if we can use the provided context to preempt this wait.
	// Unfortunately this will be tricky to implement since state tracking is
	// implemented via condition variables whereas contexts are implemented via
	// channels.
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
func (m *Manager) Pause(ctx context.Context, selection *selection.Selection, prompter string) error {
	// Extract the controllers for the sessions of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	}

	// Attempt to pause the sessions.
	for _, controller := range controllers {
		if err := controller.halt(ctx, controllerHaltModePause, prompter); err != nil {
			return errors.Wrap(err, "unable to pause session")
		}
	}

	// Success.
	return nil
}

// Resume tells the manager to resume sessions matching the given
// specifications.
func (m *Manager) Resume(ctx context.Context, selection *selection.Selection, prompter string) error {
	// Extract the controllers for the sessions of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	}

	// Attempt to resume.
	for _, controller := range controllers {
		if err := controller.resume(ctx, prompter); err != nil {
			return errors.Wrap(err, "unable to resume session")
		}
	}

	// Success.
	return nil
}

// Terminate tells the manager to terminate sessions matching the given
// specifications.
func (m *Manager) Terminate(ctx context.Context, selection *selection.Selection, prompter string) error {
	// Extract the controllers for the sessions of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return errors.Wrap(err, "unable to locate requested sessions")
	}

	// Attempt to terminate the sessions. Since we're terminating them, we're
	// responsible for removing them from the session map.
	for _, controller := range controllers {
		if err := controller.halt(ctx, controllerHaltModeTerminate, prompter); err != nil {
			return errors.Wrap(err, "unable to terminate session")
		}
		m.sessionsLock.Lock()
		delete(m.sessions, controller.session.Identifier)
		m.sessionsLock.Unlock()
	}

	// Success.
	return nil
}
