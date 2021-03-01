package forwarding

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/prompting"
	"github.com/mutagen-io/mutagen/pkg/state"
	"github.com/mutagen-io/mutagen/pkg/url"
)

const (
	// autoReconnectInterval is the period of time to wait before attempting an
	// automatic reconnect after disconnection or a failed reconnect.
	autoReconnectInterval = 15 * time.Second
)

// controller manages and executes a single session.
type controller struct {
	// logger is the controller logger.
	logger *logging.Logger
	// sessionPath is the path to the serialized session.
	sessionPath string
	// stateLock guards and tracks changes to the session member's Paused field
	// and the state member.
	stateLock *state.TrackingLock
	// session encodes the associated session metadata. It is considered static
	// and safe for concurrent access except for its Paused field, for which the
	// stateLock member should be held. It should be saved to disk any time it
	// is modified.
	session *Session
	// mergedSourceConfiguration is the source-specific configuration object
	// (computed from the core configuration and source-specific overrides). It
	// is considered static and safe for concurrent access. It is a derived
	// field and not saved to disk.
	mergedSourceConfiguration *Configuration
	// mergedDestinationConfiguration is the destination-specific configuration
	// object (computed from the core configuration and destination-specific
	// overrides). It is considered static and safe for concurrent access. It is
	// a derived field and not saved to disk.
	mergedDestinationConfiguration *Configuration
	// state represents the current forwarding state.
	state *State
	// lifecycleLock guards setting of the disabled, cancel, flushRequests, and
	// done members. Access to these members is allowed for the forwarding loop
	// without holding the lock. Any code wishing to set these members should
	// first acquire the lock, then cancel the forwarding loop, and wait for it
	// to complete before making any such changes.
	lifecycleLock sync.Mutex
	// disabled indicates that no more changes to the forwarding loop lifecycle
	// are allowed (i.e. no more forwarding loops can be started for this
	// controller). This is used by terminate and shutdown. It should only be
	// set to true once any existing forwarding loop has been stopped.
	disabled bool
	// cancel cancels the forwarding loop execution context. It should be nil if
	// and only if there is no forwarding loop running.
	cancel context.CancelFunc
	// done will be closed by the current forwarding loop when it exits.
	done chan struct{}
}

// newSession creates a new session and corresponding controller.
func newSession(
	ctx context.Context,
	logger *logging.Logger,
	tracker *state.Tracker,
	identifier string,
	source, destination *url.URL,
	configuration, configurationSource, configurationDestination *Configuration,
	name string,
	labels map[string]string,
	paused bool,
	prompter string,
) (*controller, error) {
	// Update status.
	prompting.Message(prompter, "Creating session...")

	// Set the session version.
	version := Version_Version1

	// Compute the creation time and check that it's valid for Protocol Buffers.
	creationTime := timestamppb.Now()
	if err := creationTime.CheckValid(); err != nil {
		return nil, errors.Wrap(err, "unable to record creation time")
	}

	// Compute merged endpoint configurations.
	mergedSourceConfiguration := MergeConfigurations(configuration, configurationSource)
	mergedDestinationConfiguration := MergeConfigurations(configuration, configurationDestination)

	// If the session isn't being created paused, then try to connect to the
	// endpoints. Before doing so, set up a deferred handler that will shut down
	// any endpoints that aren't handed off to the run loop due to errors.
	var sourceEndpoint, destinationEndpoint Endpoint
	var err error
	defer func() {
		if sourceEndpoint != nil {
			sourceEndpoint.Shutdown()
			sourceEndpoint = nil
		}
		if destinationEndpoint != nil {
			destinationEndpoint.Shutdown()
			destinationEndpoint = nil
		}
	}()
	if !paused {
		logger.Info("Connecting to source endpoint")
		sourceEndpoint, err = connect(
			ctx,
			logger.Sublogger("source"),
			source,
			prompter,
			identifier,
			version,
			mergedSourceConfiguration,
			true,
		)
		if err != nil {
			logger.Info("Source connection failure:", err)
			return nil, errors.Wrap(err, "unable to connect to source")
		}
		logger.Info("Connecting to destination endpoint")
		destinationEndpoint, err = connect(
			ctx,
			logger.Sublogger("destination"),
			destination,
			prompter,
			identifier,
			version,
			mergedDestinationConfiguration,
			false,
		)
		if err != nil {
			logger.Info("Destination connection failure:", err)
			return nil, errors.Wrap(err, "unable to connect to destination")
		}
	}

	// Create the session.
	session := &Session{
		Identifier:               identifier,
		Version:                  version,
		CreationTime:             creationTime,
		CreatingVersionMajor:     mutagen.VersionMajor,
		CreatingVersionMinor:     mutagen.VersionMinor,
		CreatingVersionPatch:     mutagen.VersionPatch,
		Source:                   source,
		Destination:              destination,
		Configuration:            configuration,
		ConfigurationSource:      configurationSource,
		ConfigurationDestination: configurationDestination,
		Name:                     name,
		Labels:                   labels,
		Paused:                   paused,
	}

	// Compute the session path.
	sessionPath, err := pathForSession(session.Identifier)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute session path")
	}

	// Save the session to disk.
	if err := encoding.MarshalAndSaveProtobuf(sessionPath, session); err != nil {
		return nil, errors.Wrap(err, "unable to save session")
	}

	// Create the controller.
	controller := &controller{
		logger:                         logger,
		sessionPath:                    sessionPath,
		stateLock:                      state.NewTrackingLock(tracker),
		session:                        session,
		mergedSourceConfiguration:      mergedSourceConfiguration,
		mergedDestinationConfiguration: mergedDestinationConfiguration,
		state: &State{
			Session: session,
		},
	}

	// If the session isn't being created paused, then start a forwarding loop
	// and mark the endpoints as handed off to that loop so that we don't defer
	// their shutdown.
	if !paused {
		logger.Info("Starting forwarding loop")
		ctx, cancel := context.WithCancel(context.Background())
		controller.cancel = cancel
		controller.done = make(chan struct{})
		go controller.run(ctx, sourceEndpoint, destinationEndpoint)
		sourceEndpoint = nil
		destinationEndpoint = nil
	}

	// Success.
	logger.Info("Session initialized")
	return controller, nil
}

// loadSession loads an existing session and creates a corresponding controller.
func loadSession(logger *logging.Logger, tracker *state.Tracker, identifier string) (*controller, error) {
	// Compute the session path.
	sessionPath, err := pathForSession(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute session path")
	}

	// Load and validate the session.
	session := &Session{}
	if err := encoding.LoadAndUnmarshalProtobuf(sessionPath, session); err != nil {
		return nil, errors.Wrap(err, "unable to load session configuration")
	}
	if err := session.EnsureValid(); err != nil {
		return nil, errors.Wrap(err, "invalid session found on disk")
	}

	// Create the controller.
	controller := &controller{
		logger:      logger,
		sessionPath: sessionPath,
		stateLock:   state.NewTrackingLock(tracker),
		session:     session,
		mergedSourceConfiguration: MergeConfigurations(
			session.Configuration,
			session.ConfigurationSource,
		),
		mergedDestinationConfiguration: MergeConfigurations(
			session.Configuration,
			session.ConfigurationDestination,
		),
		state: &State{
			Session: session,
		},
	}

	// If the session isn't marked as paused, start a forwarding loop.
	if !session.Paused {
		ctx, cancel := context.WithCancel(context.Background())
		controller.cancel = cancel
		controller.done = make(chan struct{})
		go controller.run(ctx, nil, nil)
	}

	// Success.
	logger.Info("Session loaded")
	return controller, nil
}

// currentState creates a snapshot of the current session state.
func (c *controller) currentState() *State {
	// Lock the session state and defer its release. It's very important that we
	// unlock without a notification here, otherwise we'd trigger an infinite
	// cycle of list/notify.
	c.stateLock.Lock()
	defer c.stateLock.UnlockWithoutNotify()

	// Perform a (pseudo) deep copy of the state.
	return c.state.copy()
}

// resume attempts to reconnect and resume the session if it isn't currently
// connected and forwarding.
func (c *controller) resume(ctx context.Context, prompter string) error {
	// Update status.
	prompting.Message(prompter, fmt.Sprintf("Resuming session %s...", c.session.Identifier))

	// Lock the controller's lifecycle and defer its release.
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Don't allow any resume operations if the controller is disabled.
	if c.disabled {
		return errors.New("controller disabled")
	}

	// Check if there's an existing forwarding loop (i.e. if the session is
	// unpaused).
	if c.cancel != nil {
		// If there is an existing forwarding loop, check if it's already in a
		// state that's considered "forwarding".
		c.stateLock.Lock()
		forwarding := c.state.Status >= Status_ForwardingConnections
		c.stateLock.UnlockWithoutNotify()

		// If we're already forwarding, then there's nothing we need to do. We
		// don't even need to mark the session as unpaused because it can't be
		// marked as paused if an existing forwarding loop is running (we
		// enforce this invariant as part of the controller's logic).
		if forwarding {
			return nil
		}

		// Otherwise, cancel the existing forwarding loop and wait for it to
		// finish.
		//
		// There's something of an efficiency race condition here, because the
		// existing loop might succeed in connecting between the time we check
		// and the time we cancel it. That could happen if an auto-reconnect
		// succeeds or even if the loop was already passed connections and it's
		// just hasn't updated its status yet. But the only danger here is
		// basically wasting those connections, and the window is very small.
		c.cancel()
		<-c.done

		// Nil out any lifecycle state.
		c.cancel = nil
		c.done = nil
	}

	// Mark the session as unpaused and save it to disk.
	c.stateLock.Lock()
	c.session.Paused = false
	saveErr := encoding.MarshalAndSaveProtobuf(c.sessionPath, c.session)
	c.stateLock.Unlock()

	// Attempt to connect to source.
	c.stateLock.Lock()
	c.state.Status = Status_ConnectingSource
	c.stateLock.Unlock()
	source, sourceConnectErr := connect(
		ctx,
		c.logger.Sublogger("source"),
		c.session.Source,
		prompter,
		c.session.Identifier,
		c.session.Version,
		c.mergedSourceConfiguration,
		true,
	)
	c.stateLock.Lock()
	c.state.SourceConnected = (source != nil)
	c.stateLock.Unlock()

	// Attempt to connect to destination.
	c.stateLock.Lock()
	c.state.Status = Status_ConnectingDestination
	c.stateLock.Unlock()
	destination, destinationConnectErr := connect(
		ctx,
		c.logger.Sublogger("destination"),
		c.session.Destination,
		prompter,
		c.session.Identifier,
		c.session.Version,
		c.mergedDestinationConfiguration,
		false,
	)
	c.stateLock.Lock()
	c.state.DestinationConnected = (destination != nil)
	c.stateLock.Unlock()

	// Start the forwarding loop with what we have. Source or destination may
	// have failed to connect (and be nil), but in any case that'll just make
	// the run loop keep trying to connect.
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.done = make(chan struct{})
	go c.run(ctx, source, destination)

	// Report any errors. Since we always want to start a forwarding loop, even
	// on partial or complete failure (since it might be able to auto-reconnect
	// on its own), we wait until the end to report errors.
	if saveErr != nil {
		return errors.Wrap(saveErr, "unable to save session")
	} else if sourceConnectErr != nil {
		return errors.Wrap(sourceConnectErr, "unable to connect to source")
	} else if destinationConnectErr != nil {
		return errors.Wrap(destinationConnectErr, "unable to connect to destination")
	}

	// Success.
	return nil
}

// controllerHaltMode represents the behavior to use when halting a session.
type controllerHaltMode uint8

const (
	// controllerHaltModePause indicates that a session should be halted and
	// marked as paused.
	controllerHaltModePause controllerHaltMode = iota
	// controllerHaltModeShutdown indicates that a session should be halted.
	controllerHaltModeShutdown
	// controllerHaltModeShutdown indicates that a session should be halted and
	// then deleted.
	controllerHaltModeTerminate
)

// description returns a human-readable description of a halt mode.
func (m controllerHaltMode) description() string {
	switch m {
	case controllerHaltModePause:
		return "Pausing"
	case controllerHaltModeShutdown:
		return "Shutting down"
	case controllerHaltModeTerminate:
		return "Terminating"
	default:
		panic("unhandled halt mode")
	}
}

// halt halts the session with the specified behavior.
func (c *controller) halt(_ context.Context, mode controllerHaltMode, prompter string) error {
	// Update status.
	prompting.Message(prompter, fmt.Sprintf("%s session %s...", mode.description(), c.session.Identifier))

	// Lock the controller's lifecycle and defer its release.
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Don't allow any additional halt operations if the controller is disabled,
	// because either this session is being terminated or the service is
	// shutting down, and in either case there is no point in halting.
	if c.disabled {
		return errors.New("controller disabled")
	}

	// Kill any existing forwarding loop.
	if c.cancel != nil {
		// Cancel the forwarding loop and wait for it to finish.
		c.cancel()
		<-c.done

		// Nil out any lifecycle state.
		c.cancel = nil
		c.done = nil
	}

	// Handle based on the halt mode.
	if mode == controllerHaltModePause {
		// Mark the session as paused and save it.
		c.stateLock.Lock()
		c.session.Paused = true
		saveErr := encoding.MarshalAndSaveProtobuf(c.sessionPath, c.session)
		c.stateLock.Unlock()
		if saveErr != nil {
			return errors.Wrap(saveErr, "unable to save session")
		}
	} else if mode == controllerHaltModeShutdown {
		// Disable the controller.
		c.disabled = true
	} else if mode == controllerHaltModeTerminate {
		// Disable the controller.
		c.disabled = true

		// Wipe the session information from disk.
		sessionRemoveErr := os.Remove(c.sessionPath)
		if sessionRemoveErr != nil {
			return errors.Wrap(sessionRemoveErr, "unable to remove session from disk")
		}
	} else {
		panic("invalid halt mode specified")
	}

	// Success.
	return nil
}

// run is the main runloop for the controller, managing connectivity and
// forwarding.
func (c *controller) run(ctx context.Context, source, destination Endpoint) {
	// Defer resource and state cleanup.
	defer func() {
		// Shutdown any endpoints. These might be non-nil if the runloop was
		// cancelled while partially connected rather than after forwarding
		// failure.
		if source != nil {
			source.Shutdown()
		}
		if destination != nil {
			destination.Shutdown()
		}

		// Reset the state.
		c.stateLock.Lock()
		c.state = &State{
			Session: c.session,
		}
		c.stateLock.Unlock()

		// Signal completion.
		close(c.done)
	}()

	// Track the last time that forwarding failed.
	var lastForwardingFailureTime time.Time

	// Loop until cancelled.
	for {
		// Loop until we're connected to both endpoints. We do a non-blocking
		// check for cancellation on each reconnect error so that we don't waste
		// resources by trying another connect when the context has been
		// cancelled (it'll be wasteful). This is better than sentinel errors.
		for {
			// Ensure that source is connected.
			var sourceConnectErr error
			if source == nil {
				c.stateLock.Lock()
				c.state.Status = Status_ConnectingSource
				c.stateLock.Unlock()
				source, sourceConnectErr = connect(
					ctx,
					c.logger.Sublogger("source"),
					c.session.Source,
					"",
					c.session.Identifier,
					c.session.Version,
					c.mergedSourceConfiguration,
					true,
				)
			}
			c.stateLock.Lock()
			c.state.SourceConnected = (source != nil)
			if sourceConnectErr != nil {
				c.state.LastError = errors.Wrap(sourceConnectErr, "unable to connect to source").Error()
			}
			c.stateLock.Unlock()

			// Check for cancellation to avoid a spurious connection to
			// destination in case cancellation occurred while connecting to
			// source.
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Ensure that destination is connected.
			var destinationConnectErr error
			if destination == nil {
				c.stateLock.Lock()
				c.state.Status = Status_ConnectingDestination
				c.stateLock.Unlock()
				destination, destinationConnectErr = connect(
					ctx,
					c.logger.Sublogger("destination"),
					c.session.Destination,
					"",
					c.session.Identifier,
					c.session.Version,
					c.mergedDestinationConfiguration,
					false,
				)
			}
			c.stateLock.Lock()
			c.state.DestinationConnected = (destination != nil)
			if destinationConnectErr != nil {
				c.state.LastError = errors.Wrap(destinationConnectErr, "unable to connect to destination").Error()
			}
			c.stateLock.Unlock()

			// If both endpoints are connected, we're done. We perform this
			// check here (rather than in the loop condition) because if we did
			// it in the loop condition we'd still need a check here to avoid a
			// sleep every time (even if already successfully connected).
			if source != nil && destination != nil {
				break
			}

			// If we failed to connect, wait and then retry. Watch for
			// cancellation in the mean time.
			select {
			case <-ctx.Done():
				return
			case <-time.After(autoReconnectInterval):
			}
		}

		// Grab transport error channels for each endpoint.
		sourceTransportErrors := source.TransportErrors()
		destinationTransportErrors := destination.TransportErrors()

		// Create a cancellable subcontext that we can use to manage shutdown.
		shutdownCtx, forceShutdown := context.WithCancel(ctx)

		// Create a Goroutine that will shut down (and unblock) endpoints. This
		// is the only way to unblock forwarding on cancellation.
		shutdownComplete := make(chan struct{})
		go func() {
			<-shutdownCtx.Done()
			source.Shutdown()
			destination.Shutdown()
			close(shutdownComplete)
		}()

		// Perform forwarding in a background Goroutine and monitor for errors.
		forwardingErrors := make(chan error, 1)
		go func() {
			forwardingErrors <- c.forward(source, destination)
		}()

		// Wait for cancellation, an error from forwarding, or an error from
		// either transport.
		var sessionErr error
		var forwardingErrorReceived bool
		select {
		case <-ctx.Done():
			sessionErr = errors.New("session cancelled")
		case sessionErr = <-forwardingErrors:
			forwardingErrorReceived = true
		case err := <-sourceTransportErrors:
			sessionErr = fmt.Errorf("source transport failure: %w", err)
		case err := <-destinationTransportErrors:
			sessionErr = fmt.Errorf("destination transport failure: %w", err)
		}

		// Force shutdown, which may have already occurred due to cancellation.
		forceShutdown()

		// Wait for shutdown to complete.
		<-shutdownComplete

		// If the forwarding loop wasn't what unblocked our wait, then wait for
		// it to return a result so that we know it has exited. This isn't
		// strictly necessary with our current design, but it's cleaner and more
		// robust.
		if !forwardingErrorReceived {
			<-forwardingErrors
		}

		// Nil out endpoints to update our state.
		source = nil
		destination = nil

		// Reset the forwarding state, but propagate the error that caused
		// failure.
		c.stateLock.Lock()
		c.state = &State{
			Session:   c.session,
			LastError: sessionErr.Error(),
		}
		c.stateLock.Unlock()

		// When forwarding fails, we generally want to restart it as quickly as
		// possible. Thus, if it's been longer than our usual waiting period
		// since forwarding failed last, simply try to reconnect immediately
		// (though still check for cancellation). If it's been less than our
		// usual waiting period since forwarding failed last, then something is
		// probably wrong, so wait for our usual waiting period (while checking
		// and monitoring for cancellation).
		now := time.Now()
		if now.Sub(lastForwardingFailureTime) >= autoReconnectInterval {
			select {
			case <-ctx.Done():
				return
			default:
			}
		} else {
			select {
			case <-ctx.Done():
				return
			case <-time.After(autoReconnectInterval):
			}
		}
		lastForwardingFailureTime = now
	}
}

// forward is the main forwarding loop for the controller.
func (c *controller) forward(source, destination Endpoint) error {
	// Create a context that we can use to regulate the lifecycle of forwarding
	// Goroutines and defer its cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Clear any error state and update the status to forwarding. While we're at
	// it, capture a pointer to the state instance that all forwarding
	// Goroutines spawned by this loop will update. This state instance will be
	// replaced once this loop returns, so those background Goroutines can
	// continue to safely update it without any risk of updating a future loop's
	// state object. The only penalty is that both state objects will share the
	// same lock, but that's a negligible overhead.
	var state *State
	c.stateLock.Lock()
	c.state.LastError = ""
	c.state.Status = Status_ForwardingConnections
	state = c.state
	c.stateLock.Unlock()

	// Accept and forward connections until there's an error.
	for {
		// Accept a connection from the source.
		incoming, err := source.Open()
		if err != nil {
			return errors.Wrap(err, "unable to accept connection")
		}

		// Open the outgoing connection to which we should forward.
		outgoing, err := destination.Open()
		if err != nil {
			incoming.Close()
			return errors.Wrap(err, "unable to open forwarding connection")
		}

		// Increment the open and total connection counts.
		c.stateLock.Lock()
		state.OpenConnections++
		state.TotalConnections++
		c.stateLock.Unlock()

		// Perform forwarding and update state in a background Goroutine.
		go func() {
			// Perform forwarding.
			ForwardAndClose(ctx, incoming, outgoing)

			// Decrement open connection counts.
			c.stateLock.Lock()
			state.OpenConnections--
			c.stateLock.Unlock()
		}()
	}
}
