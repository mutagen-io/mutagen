package tunneling

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/selection"
	"github.com/mutagen-io/mutagen/pkg/state"
)

// Manager provides forwarding tunnel management facilities. Its methods are
// safe for concurrent usage, so it can be easily exported via an RPC interface.
type Manager struct {
	// logger is the underlying logger.
	logger *logging.Logger
	// tracker tracks changes to tunnel states.
	tracker *state.Tracker
	// tunnelsLock locks the tunnels registry.
	tunnelsLock *state.TrackingLock
	// tunnels maps tunnel identifiers to their respective controllers.
	tunnels map[string]*controller
}

// NewManager creates a new Manager instance.
func NewManager(logger *logging.Logger) (*Manager, error) {
	// Create a tracker and corresponding lock to watch for state changes.
	tracker := state.NewTracker()
	tunnelsLock := state.NewTrackingLock(tracker)

	// Create the tunnel registry.
	tunnels := make(map[string]*controller)

	// Load existing tunnels.
	logger.Info("Looking for existing tunnels")
	tunnelsDirectory, err := pathForTunnel("")
	if err != nil {
		return nil, fmt.Errorf("unable to compute tunnels directory: %w", err)
	}
	tunnelsDirectoryContents, err := filesystem.DirectoryContentsByPath(tunnelsDirectory)
	if err != nil {
		return nil, fmt.Errorf("unable to read contents of tunnels directory: %w", err)
	}
	for _, c := range tunnelsDirectoryContents {
		// TODO: Ensure that the name matches the expected format.
		identifier := c.Name()
		logger.Info("Loading tunnel", identifier)
		if controller, err := loadTunnel(logger.Sublogger(identifier), tracker, identifier); err != nil {
			continue
		} else {
			tunnels[identifier] = controller
		}
	}

	// Success.
	logger.Info("Tunnel manager initialized")
	return &Manager{
		logger:      logger,
		tracker:     tracker,
		tunnelsLock: tunnelsLock,
		tunnels:     tunnels,
	}, nil
}

// allControllers creates a list of all controllers managed by the manager.
func (m *Manager) allControllers() []*controller {
	// Grab the registry lock and defer its release.
	m.tunnelsLock.Lock()
	defer m.tunnelsLock.UnlockWithoutNotify()

	// Generate a list of all controllers.
	controllers := make([]*controller, 0, len(m.tunnels))
	for _, controller := range m.tunnels {
		controllers = append(controllers, controller)
	}

	// Done.
	return controllers
}

// findControllersBySpecification generates a list of controllers matching the
// given specifications.
func (m *Manager) findControllersBySpecification(specifications []string) ([]*controller, error) {
	// Grab the registry lock and defer its release.
	m.tunnelsLock.Lock()
	defer m.tunnelsLock.UnlockWithoutNotify()

	// Generate a list of controllers matching the specifications. We allow each
	// specification to match multiple controllers, so we store matches in a set
	// before converting them to a list. We do require that each specification
	// match at least one controller.
	controllerSet := make(map[*controller]bool)
	for _, specification := range specifications {
		var matched bool
		for _, controller := range m.tunnels {
			if controller.tunnel.Identifier == specification || controller.tunnel.Name == specification {
				controllerSet[controller] = true
				matched = true
			}
		}
		if !matched {
			return nil, fmt.Errorf("specification \"%s\" did not match any tunnels", specification)
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
		return nil, fmt.Errorf("unable to parse label selector: %w", err)
	}

	// Grab the registry lock and defer its release.
	m.tunnelsLock.Lock()
	defer m.tunnelsLock.UnlockWithoutNotify()

	// Loop over controllers and look for matches.
	var controllers []*controller
	for _, controller := range m.tunnels {
		if selector.Matches(controller.tunnel.Labels) {
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
		return nil, errors.New("invalid tunnel selection")
	}
}

// Shutdown tells the manager to gracefully halt tunnels.
func (m *Manager) Shutdown() {
	// Log the shutdown.
	m.logger.Info("Shutting down")

	// Poison state tracking to terminate monitoring.
	m.tracker.Poison()

	// Grab the registry lock and defer its release.
	m.tunnelsLock.Lock()
	defer m.tunnelsLock.UnlockWithoutNotify()

	// Attempt to halt each tunnel so that it can shutdown cleanly. Ignore but
	// log any that fail to halt.
	for _, controller := range m.tunnels {
		m.logger.Info("Halting tunnel", controller.tunnel.Identifier)
		if err := controller.halt(context.Background(), controllerHaltModeShutdown, ""); err != nil {
			// TODO: Log this halt failure.
		}
	}
}

// Dial performs a dial operation on the tunnel specified by identifierOrName.
// This dial invokes an agent binary in the specified mode with a version
// compatible with the current Mutagen version.
func (m *Manager) Dial(
	ctx context.Context,
	identifierOrName string,
	mode string,
	prompter string,
) (net.Conn, error) {
	// Create a selection to identify the relevant controller.
	selection := &selection.Selection{
		Specifications: []string{identifierOrName},
	}

	// Locate the relevant controller.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return nil, fmt.Errorf("unable to locate requested tunnel: %w", err)
	} else if len(controllers) != 1 {
		return nil, errors.New("tunnel specification is ambiguous")
	}

	// Perform the dial operation.
	return controllers[0].dial(ctx, mode)
}

// Create tells the manager to create a new tunnel.
func (m *Manager) Create(
	ctx context.Context,
	configuration *Configuration,
	name string,
	labels map[string]string,
	paused bool,
	prompter string,
) (*TunnelHostCredentials, error) {
	// Attempt to create a tunnel.
	// TODO: Can we create a (meaningful) sublogger here? We don't know the
	// tunnel identifier. I suppose we could use the tunnel name.
	controller, hostCredentials, err := newTunnel(
		ctx,
		m.logger,
		m.tracker,
		configuration,
		name,
		labels,
		paused,
		prompter,
	)
	if err != nil {
		return nil, err
	}

	// Register the controller.
	m.tunnelsLock.Lock()
	m.tunnels[controller.tunnel.Identifier] = controller
	m.tunnelsLock.Unlock()

	// Done.
	return hostCredentials, nil
}

// List requests a state snapshot for the specified tunnels.
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

	// Extract the controllers for the tunnels of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return 0, nil, fmt.Errorf("unable to locate requested tunnels: %w", err)
	}

	// Extract the state from each controller.
	states := make([]*State, len(controllers))
	for i, controller := range controllers {
		states[i] = controller.currentState()
	}

	// Sort tunnel states by tunnel creation time.
	sort.Slice(states, func(i, j int) bool {
		iTime := states[i].Tunnel.CreationTime
		jTime := states[j].Tunnel.CreationTime
		return iTime.Seconds < jTime.Seconds ||
			(iTime.Seconds == jTime.Seconds && iTime.Nanos < jTime.Nanos)
	})

	// Success.
	return stateIndex, states, nil
}

// Pause tells the manager to pause tunnels matching the given specifications.
func (m *Manager) Pause(ctx context.Context, selection *selection.Selection, prompter string) error {
	// Extract the controllers for the tunnels of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return fmt.Errorf("unable to locate requested tunnels: %w", err)
	}

	// Attempt to pause the tunnels.
	for _, controller := range controllers {
		if err := controller.halt(ctx, controllerHaltModePause, prompter); err != nil {
			return fmt.Errorf("unable to pause tunnel: %w", err)
		}
	}

	// Success.
	return nil
}

// Resume tells the manager to resume tunnels matching the given specifications.
func (m *Manager) Resume(ctx context.Context, selection *selection.Selection, prompter string) error {
	// Extract the controllers for the tunnels of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return fmt.Errorf("unable to locate requested tunnels: %w", err)
	}

	// Attempt to resume.
	for _, controller := range controllers {
		if err := controller.resume(ctx, prompter); err != nil {
			return fmt.Errorf("unable to resume tunnel: %w", err)
		}
	}

	// Success.
	return nil
}

// Terminate tells the manager to terminate tunnels matching the given
// specifications.
func (m *Manager) Terminate(ctx context.Context, selection *selection.Selection, prompter string) error {
	// Extract the controllers for the tunnels of interest.
	controllers, err := m.selectControllers(selection)
	if err != nil {
		return fmt.Errorf("unable to locate requested tunnels: %w", err)
	}

	// Attempt to terminate the tunnels. Since we're terminating them, we're
	// responsible for removing them from the tunnel map.
	for _, controller := range controllers {
		if err := controller.halt(ctx, controllerHaltModeTerminate, prompter); err != nil {
			return fmt.Errorf("unable to terminate tunnel: %w", err)
		}
		m.tunnelsLock.Lock()
		delete(m.tunnels, controller.tunnel.Identifier)
		m.tunnelsLock.Unlock()
	}

	// Success.
	return nil
}
