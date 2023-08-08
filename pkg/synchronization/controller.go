package synchronization

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/prompting"
	"github.com/mutagen-io/mutagen/pkg/state"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore"
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
	"github.com/mutagen-io/mutagen/pkg/url"
)

const (
	// autoReconnectInterval is the period of time to wait before attempting an
	// automatic reconnect after disconnection or a failed reconnect.
	autoReconnectInterval = 15 * time.Second
	// rescanWaitDuration is the period of time to wait before attempting to
	// rescan after an ephemeral scan failure.
	rescanWaitDuration = 5 * time.Second
)

// controller manages and executes a single session.
type controller struct {
	// logger is the controller logger.
	logger *logging.Logger
	// sessionPath is the path to the serialized session.
	sessionPath string
	// archivePath is the path to the serialized archive.
	archivePath string
	// stateLock guards and tracks changes to session's Paused field, state, and
	// synchronizing. Previous holders may continue to poll on synchronizing if
	// they store it in a separate variable before releasing the lock.
	stateLock *state.TrackingLock
	// session encodes the associated session metadata. It is considered static
	// and safe for concurrent access except for its Paused field, for which
	// stateLock should be held. It should be saved to disk any time it is
	// modified.
	session *Session
	// mergedAlphaConfiguration is the alpha-specific configuration object
	// (computed from the core configuration and alpha-specific overrides). It
	// is considered static and safe for concurrent access. It is a derived
	// field and not saved to disk.
	mergedAlphaConfiguration *Configuration
	// mergedBetaConfiguration is the beta-specific configuration object
	// (computed from the core configuration and beta-specific overrides). It is
	// considered static and safe for concurrent access. It is a derived field
	// and not saved to disk.
	mergedBetaConfiguration *Configuration
	// state represents the current synchronization state.
	state *State
	// synchronizing is used to track whether or not the synchronization loop is
	// currently in a state where it is capable of performing synchronization.
	// It is non-nil if and only if the synchronization loop is connected and in
	// a state where it can perform synchronization. It is closed when
	// synchronization fails due to an error.
	synchronizing chan struct{}
	// lifecycleLock guards access to disabled, cancel, flushRequests, and done.
	// Only the current holder of the lifecycle lock may set any of these fields
	// or invoke cancel. The synchronization loop may close close done or
	// receive from flushRequests without holding the lifecycle lock. Moreover,
	// previous lifecycle lock holders may continue to send to flushRequests and
	// poll on done after storing them in separate variables and releasing the
	// lifecycle lock. Any code wishing to set these fields must first acquire
	// the lock, then cancel the synchronization loop and wait for it to
	// complete before making any changes.
	lifecycleLock sync.Mutex
	// disabled indicates that no more changes to the synchronization loop
	// lifecycle are allowed (i.e. no more synchronization loops can be started
	// for this controller). This is used by terminate and shutdown. It should
	// only be set to true once any existing synchronization loop has been
	// stopped.
	disabled bool
	// cancel cancels the synchronization loop execution context. It is nil if
	// and only if there is no synchronization loop running.
	cancel context.CancelFunc
	// flushRequests is used pass flush requests to the synchronization loop. It
	// is buffered, allowing a single request to be queued. All requests passed
	// via this channel must be buffered and contain room for one error.
	flushRequests chan chan error
	// done will be closed by the current synchronization loop when it exits.
	done chan struct{}
}

// newSession creates a new session and corresponding controller.
func newSession(
	ctx context.Context,
	logger *logging.Logger,
	tracker *state.Tracker,
	identifier string,
	alpha, beta *url.URL,
	configuration, configurationAlpha, configurationBeta *Configuration,
	name string,
	labels map[string]string,
	paused bool,
	prompter string,
) (*controller, error) {
	// Update status.
	prompting.Message(prompter, "Creating session...")

	// Set the session version.
	version := DefaultVersion

	// Compute the creation time and check that it's valid for Protocol Buffers.
	creationTime := timestamppb.Now()
	if err := creationTime.CheckValid(); err != nil {
		return nil, fmt.Errorf("unable to record creation time: %w", err)
	}

	// Compute merged endpoint configurations.
	mergedAlphaConfiguration := MergeConfigurations(configuration, configurationAlpha)
	mergedBetaConfiguration := MergeConfigurations(configuration, configurationBeta)

	// If the session isn't being created paused, then try to connect to the
	// endpoints. Before doing so, set up a deferred handler that will shut down
	// any endpoints that aren't handed off to the run loop due to errors.
	var alphaEndpoint, betaEndpoint Endpoint
	var err error
	defer func() {
		if alphaEndpoint != nil {
			alphaEndpoint.Shutdown()
			alphaEndpoint = nil
		}
		if betaEndpoint != nil {
			betaEndpoint.Shutdown()
			betaEndpoint = nil
		}
	}()
	if !paused {
		logger.Info("Connecting to alpha endpoint")
		alphaEndpoint, err = connect(
			ctx,
			logger.Sublogger("alpha"),
			alpha,
			prompter,
			identifier,
			version,
			mergedAlphaConfiguration,
			true,
		)
		if err != nil {
			logger.Info("Alpha connection failure:", err)
			return nil, fmt.Errorf("unable to connect to alpha: %w", err)
		}
		logger.Info("Connecting to beta endpoint")
		betaEndpoint, err = connect(
			ctx,
			logger.Sublogger("beta"),
			beta,
			prompter,
			identifier,
			version,
			mergedBetaConfiguration,
			false,
		)
		if err != nil {
			logger.Info("Beta connection failure:", err)
			return nil, fmt.Errorf("unable to connect to beta: %w", err)
		}
	}

	// Create the session and initial archive.
	session := &Session{
		Identifier:           identifier,
		Version:              version,
		CreationTime:         creationTime,
		CreatingVersionMajor: mutagen.VersionMajor,
		CreatingVersionMinor: mutagen.VersionMinor,
		CreatingVersionPatch: mutagen.VersionPatch,
		Alpha:                alpha,
		Beta:                 beta,
		Configuration:        configuration,
		ConfigurationAlpha:   configurationAlpha,
		ConfigurationBeta:    configurationBeta,
		Name:                 name,
		Labels:               labels,
		Paused:               paused,
	}
	archive := &core.Archive{}

	// Compute the session and archive paths.
	sessionPath, err := pathForSession(session.Identifier)
	if err != nil {
		return nil, fmt.Errorf("unable to compute session path: %w", err)
	}
	archivePath, err := pathForArchive(session.Identifier)
	if err != nil {
		return nil, fmt.Errorf("unable to compute archive path: %w", err)
	}

	// Save components to disk.
	if err := encoding.MarshalAndSaveProtobuf(sessionPath, session); err != nil {
		return nil, fmt.Errorf("unable to save session: %w", err)
	}
	if err := encoding.MarshalAndSaveProtobuf(archivePath, archive); err != nil {
		os.Remove(sessionPath)
		return nil, fmt.Errorf("unable to save archive: %w", err)
	}

	// Create the controller.
	controller := &controller{
		logger:                   logger,
		sessionPath:              sessionPath,
		archivePath:              archivePath,
		stateLock:                state.NewTrackingLock(tracker),
		session:                  session,
		mergedAlphaConfiguration: mergedAlphaConfiguration,
		mergedBetaConfiguration:  mergedBetaConfiguration,
		state: &State{
			Session:    session,
			AlphaState: &EndpointState{},
			BetaState:  &EndpointState{},
		},
	}

	// If the session isn't being created paused, then start a synchronization
	// loop and mark the endpoints as handed off to that loop so that we don't
	// defer their shutdown.
	if !paused {
		ctx, cancel := context.WithCancel(context.Background())
		controller.cancel = cancel
		controller.flushRequests = make(chan chan error, 1)
		controller.done = make(chan struct{})
		go controller.run(ctx, alphaEndpoint, betaEndpoint)
		alphaEndpoint = nil
		betaEndpoint = nil
	}

	// Success.
	logger.Info("Session initialized")
	return controller, nil
}

// loadSession loads an existing session and creates a corresponding controller.
func loadSession(logger *logging.Logger, tracker *state.Tracker, identifier string) (*controller, error) {
	// Compute session and archive paths.
	sessionPath, err := pathForSession(identifier)
	if err != nil {
		return nil, fmt.Errorf("unable to compute session path: %w", err)
	}
	archivePath, err := pathForArchive(identifier)
	if err != nil {
		return nil, fmt.Errorf("unable to compute archive path: %w", err)
	}

	// Load and validate the session. We have to populate a few optional fields
	// before validation if they're not set. We can't do this in the Session
	// literal because they'll be wiped out during unmarshalling, even if not
	// set.
	session := &Session{}
	if err := encoding.LoadAndUnmarshalProtobuf(sessionPath, session); err != nil {
		return nil, fmt.Errorf("unable to load session configuration: %w", err)
	}
	if session.ConfigurationAlpha == nil {
		session.ConfigurationAlpha = &Configuration{}
	}
	if session.ConfigurationBeta == nil {
		session.ConfigurationBeta = &Configuration{}
	}
	if err := session.EnsureValid(); err != nil {
		return nil, fmt.Errorf("invalid session found on disk: %w", err)
	}

	// Create the controller.
	controller := &controller{
		logger:      logger,
		sessionPath: sessionPath,
		archivePath: archivePath,
		stateLock:   state.NewTrackingLock(tracker),
		session:     session,
		mergedAlphaConfiguration: MergeConfigurations(
			session.Configuration,
			session.ConfigurationAlpha,
		),
		mergedBetaConfiguration: MergeConfigurations(
			session.Configuration,
			session.ConfigurationBeta,
		),
		state: &State{
			Session:    session,
			AlphaState: &EndpointState{},
			BetaState:  &EndpointState{},
		},
	}

	// If the session isn't marked as paused, start a synchronization loop.
	if !session.Paused {
		ctx, cancel := context.WithCancel(context.Background())
		controller.cancel = cancel
		controller.flushRequests = make(chan chan error, 1)
		controller.done = make(chan struct{})
		go controller.run(ctx, nil, nil)
	}

	// Success.
	logger.Info("Session loaded")
	return controller, nil
}

// currentState creates a static snapshot of the current session state.
func (c *controller) currentState() *State {
	// Lock the session state and defer its release. It's very important that we
	// unlock without a notification here, otherwise we'd trigger an infinite
	// cycle of list/notify.
	c.stateLock.Lock()
	defer c.stateLock.UnlockWithoutNotify()

	// Create a static copy of the state.
	return proto.Clone(c.state).(*State)
}

// flush attempts to force a synchronization cycle for the session. If wait is
// specified, then the method will wait until a post-flush synchronization cycle
// has completed. The provided context (which must be non-nil) can terminate
// this wait early.
func (c *controller) flush(ctx context.Context, prompter string, skipWait bool) error {
	// Update status.
	prompting.Message(prompter, fmt.Sprintf("Forcing synchronization cycle for session %s...", c.session.Identifier))

	// Lock the controller's lifecycle.
	c.lifecycleLock.Lock()

	// Don't allow any operations if the controller is disabled.
	if c.disabled {
		c.lifecycleLock.Unlock()
		return errors.New("controller disabled")
	}

	// Check if the session is paused.
	if c.cancel == nil {
		c.lifecycleLock.Unlock()
		return errors.New("session is paused")
	}

	// Perform logging.
	c.logger.Infof("Forcing synchronization cycle")

	// Check if the session is currently synchronizing and store the channel
	// that we'll use to track synchronizability.
	c.stateLock.Lock()
	synchronizing := c.synchronizing
	c.stateLock.UnlockWithoutNotify()
	if synchronizing == nil {
		c.lifecycleLock.Unlock()
		return errors.New("session is not currently able to synchronize")
	}

	// Store the channels that we'll need to submit flush requests and track
	// synchronization termination.
	flushRequests := c.flushRequests
	done := c.done

	// Release the lifecycle lock.
	c.lifecycleLock.Unlock()

	// Create a flush request.
	request := make(chan error, 1)

	// If we don't want to wait, then we can simply send the request in a
	// non-blocking manner, in which case either this request (or one that's
	// already queued) will be processed eventually. After that, we're done. In
	// this case, we'll still check for an inability to synchronize, since we
	// may as well report it if we can.
	if skipWait {
		select {
		case flushRequests <- request:
			return nil
		case <-synchronizing:
			return errors.New("synchronization failed before flush request could be sent")
		case <-done:
			return errors.New("synchronization terminated before flush request could be sent")
		default:
			return nil
		}
	}

	// Otherwise we need to send the request in a blocking manner, watching for
	// cancellation, failure, or termination.
	select {
	case flushRequests <- request:
	case <-ctx.Done():
		return errors.New("flush cancelled before request could be sent")
	case <-synchronizing:
		return errors.New("synchronization failed before flush request could be sent")
	case <-done:
		return errors.New("synchronization terminated before flush request could be sent")
	}

	// Now we need to wait for a response to the request, again watching for
	// cancellation, failure, or termination.
	select {
	case err := <-request:
		return err
	case <-ctx.Done():
		return errors.New("flush cancelled while waiting for response")
	case <-synchronizing:
		return errors.New("synchronization failed while waiting for flush response")
	case <-done:
		return errors.New("synchronization terminated while waiting for flush response")
	}
}

// resume attempts to reconnect and resume the session if it isn't currently
// connected and synchronizing. If lifecycleLockHeld is true, then halt will
// assume that the lifecycle lock is held by the caller and will not attempt to
// acquire it.
func (c *controller) resume(ctx context.Context, prompter string, lifecycleLockHeld bool) error {
	// Update status.
	prompting.Message(prompter, fmt.Sprintf("Resuming session %s...", c.session.Identifier))

	// If not already held, acquire the lifecycle lock and defer its release.
	if !lifecycleLockHeld {
		c.lifecycleLock.Lock()
		defer c.lifecycleLock.Unlock()
	}

	// Don't allow any resume operations if the controller is disabled.
	if c.disabled {
		return errors.New("controller disabled")
	}

	// Perform logging.
	c.logger.Infof("Resuming")

	// Check if there's an existing synchronization loop (i.e. if the session is
	// unpaused).
	if c.cancel != nil {
		// If there is an existing synchronization loop, check if it's already
		// in a state that's considered "connected".
		c.stateLock.Lock()
		connected := c.state.Status >= Status_Watching
		c.stateLock.UnlockWithoutNotify()

		// If we're already connected, then there's nothing we need to do. We
		// don't even need to mark the session as unpaused because it can't be
		// marked as paused if an existing synchronization loop is running (we
		// enforce this invariant as part of the controller's logic).
		if connected {
			return nil
		}

		// Otherwise, cancel the existing synchronization loop and wait for it
		// to finish.
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
		c.flushRequests = nil
		c.done = nil
	}

	// Mark the session as unpaused and save it to disk.
	c.stateLock.Lock()
	c.session.Paused = false
	saveErr := encoding.MarshalAndSaveProtobuf(c.sessionPath, c.session)
	c.stateLock.Unlock()

	// Attempt to connect to alpha.
	c.stateLock.Lock()
	c.state.Status = Status_ConnectingAlpha
	c.stateLock.Unlock()
	alpha, alphaConnectErr := connect(
		ctx,
		c.logger.Sublogger("alpha"),
		c.session.Alpha,
		prompter,
		c.session.Identifier,
		c.session.Version,
		c.mergedAlphaConfiguration,
		true,
	)
	c.stateLock.Lock()
	c.state.AlphaState.Connected = (alpha != nil)
	c.stateLock.Unlock()

	// Attempt to connect to beta.
	c.stateLock.Lock()
	c.state.Status = Status_ConnectingBeta
	c.stateLock.Unlock()
	beta, betaConnectErr := connect(
		ctx,
		c.logger.Sublogger("beta"),
		c.session.Beta,
		prompter,
		c.session.Identifier,
		c.session.Version,
		c.mergedBetaConfiguration,
		false,
	)
	c.stateLock.Lock()
	c.state.BetaState.Connected = (beta != nil)
	c.stateLock.Unlock()

	// Start the synchronization loop with what we have. Alpha or beta may have
	// failed to connect (and be nil), but in any case that'll just make the run
	// loop keep trying to connect.
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.flushRequests = make(chan chan error, 1)
	c.done = make(chan struct{})
	go c.run(ctx, alpha, beta)

	// Report any errors. Since we always want to start a synchronization loop,
	// even on partial or complete failure (since it might be able to
	// auto-reconnect on its own), we wait until the end to report errors.
	if saveErr != nil {
		return fmt.Errorf("unable to save session: %w", saveErr)
	} else if alphaConnectErr != nil {
		return fmt.Errorf("unable to connect to alpha: %w", alphaConnectErr)
	} else if betaConnectErr != nil {
		return fmt.Errorf("unable to connect to beta: %w", betaConnectErr)
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

// halt halts the session with the specified behavior. If lifecycleLockHeld is
// true, then halt will assume that the lifecycle lock is held by the caller and
// will not attempt to acquire it.
func (c *controller) halt(_ context.Context, mode controllerHaltMode, prompter string, lifecycleLockHeld bool) error {
	// Update status.
	prompting.Message(prompter, fmt.Sprintf("%s session %s...", mode.description(), c.session.Identifier))

	// If not already held, acquire the lifecycle lock and defer its release.
	if !lifecycleLockHeld {
		c.lifecycleLock.Lock()
		defer c.lifecycleLock.Unlock()
	}

	// Don't allow any additional halt operations if the controller is disabled,
	// because either this session is being terminated or the service is
	// shutting down, and in either case there is no point in halting.
	if c.disabled {
		return errors.New("controller disabled")
	}

	// Perform logging.
	c.logger.Infof(mode.description())

	// Kill any existing synchronization loop.
	if c.cancel != nil {
		// Cancel the synchronization loop and wait for it to finish.
		c.cancel()
		<-c.done

		// Nil out any lifecycle state.
		c.cancel = nil
		c.flushRequests = nil
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
			return fmt.Errorf("unable to save session: %w", saveErr)
		}
	} else if mode == controllerHaltModeShutdown {
		// Disable the controller.
		c.disabled = true
	} else if mode == controllerHaltModeTerminate {
		// Disable the controller.
		c.disabled = true

		// Wipe the session information from disk.
		sessionRemoveErr := os.Remove(c.sessionPath)
		archiveRemoveErr := os.Remove(c.archivePath)
		if sessionRemoveErr != nil {
			return fmt.Errorf("unable to remove session from disk: %w", sessionRemoveErr)
		} else if archiveRemoveErr != nil {
			return fmt.Errorf("unable to remove archive from disk: %w", archiveRemoveErr)
		}
	} else {
		panic("invalid halt mode specified")
	}

	// Success.
	return nil
}

// reset resets synchronization session history by pausing the session (if it's
// running), overwriting the ancestor data stored on disk with an empty
// ancestor, and then resuming the session (if it was previously running).
func (c *controller) reset(ctx context.Context, prompter string) error {
	// Lock the controller's lifecycle and defer its release.
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Check if the session is currently running.
	running := c.cancel != nil

	// If the session is running, pause it.
	if running {
		if err := c.halt(ctx, controllerHaltModePause, prompter, true); err != nil {
			return fmt.Errorf("unable to pause session: %w", err)
		}
	}

	// Reset the session archive on disk.
	c.logger.Infof("Resetting ancestor")
	archive := &core.Archive{}
	if err := encoding.MarshalAndSaveProtobuf(c.archivePath, archive); err != nil {
		return fmt.Errorf("unable to clear session history: %w", err)
	}

	// Resume the session if it was previously running.
	if running {
		if err := c.resume(ctx, prompter, true); err != nil {
			return fmt.Errorf("unable to resume session: %w", err)
		}
	}

	// Success.
	return nil
}

var (
	// errHaltedForSafety is a sentinel error indicating that a safety check
	// wants the synchronization loop to be halted until manually resumed.
	errHaltedForSafety = errors.New("synchronization halted")
)

// run is the main run loop for the controller, managing connectivity and
// synchronization.
func (c *controller) run(ctx context.Context, alpha, beta Endpoint) {
	// Log run loop entry.
	c.logger.Debug("Run loop commencing")

	// Defer resource and state cleanup.
	defer func() {
		// Shutdown any endpoints. These might be non-nil if the run loop was
		// cancelled while partially connected rather than after sync failure.
		if alpha != nil {
			alpha.Shutdown()
		}
		if beta != nil {
			beta.Shutdown()
		}

		// Reset the state.
		c.stateLock.Lock()
		c.state = &State{
			Session:    c.session,
			AlphaState: &EndpointState{},
			BetaState:  &EndpointState{},
		}
		c.stateLock.Unlock()

		// Log run loop termination.
		c.logger.Debug("Run loop terminated")

		// Signal completion.
		close(c.done)
	}()

	// Track the last time that synchronization failed.
	var lastSynchronizationFailureTime time.Time

	// Loop until cancelled.
	for {
		// Loop until we're connected to both endpoints. We do a non-blocking
		// check for cancellation on each reconnect error so that we don't waste
		// resources by trying another connect when the context has been
		// cancelled (it'll be wasteful). This is better than sentinel errors.
		for {
			// Ensure that alpha is connected.
			if alpha == nil {
				c.stateLock.Lock()
				c.state.Status = Status_ConnectingAlpha
				c.stateLock.Unlock()
				alpha, _ = connect(
					ctx,
					c.logger.Sublogger("alpha"),
					c.session.Alpha,
					"",
					c.session.Identifier,
					c.session.Version,
					c.mergedAlphaConfiguration,
					true,
				)
			}
			c.stateLock.Lock()
			c.state.AlphaState.Connected = (alpha != nil)
			c.stateLock.Unlock()

			// Check for cancellation to avoid a spurious connection to beta in
			// case cancellation occurred while connecting to alpha.
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Ensure that beta is connected.
			if beta == nil {
				c.stateLock.Lock()
				c.state.Status = Status_ConnectingBeta
				c.stateLock.Unlock()
				beta, _ = connect(
					ctx,
					c.logger.Sublogger("beta"),
					c.session.Beta,
					"",
					c.session.Identifier,
					c.session.Version,
					c.mergedBetaConfiguration,
					false,
				)
			}
			c.stateLock.Lock()
			c.state.BetaState.Connected = (beta != nil)
			c.stateLock.Unlock()

			// If both endpoints are connected, we're done. We perform this
			// check here (rather than in the loop condition) because if we did
			// it in the loop condition we'd still need a check here to avoid a
			// sleep every time (even if already successfully connected).
			if alpha != nil && beta != nil {
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

		// Indicate that the synchronization loop is entering a state where it
		// can actually perform synchronization. We don't need to perform any
		// notification here since this is not a user-visible state change.
		c.stateLock.Lock()
		c.synchronizing = make(chan struct{})
		c.stateLock.UnlockWithoutNotify()

		// Perform synchronization.
		c.logger.Debug("Entering synchronization loop")
		err := c.synchronize(ctx, alpha, beta)
		c.logger.Debug("Synchronization loop terminated with error:", err)

		// Indicate that the synchronization loop is no longer synchronizing.
		// Again, no notification is required here since this is not a
		// user-visible state change.
		c.stateLock.Lock()
		close(c.synchronizing)
		c.synchronizing = nil
		c.stateLock.UnlockWithoutNotify()

		// Shutdown the endpoints.
		alpha.Shutdown()
		alpha = nil
		beta.Shutdown()
		beta = nil

		// If synchronization failed due a halting error, then wait for the
		// synchronization loop to be manually resumed.
		if err == errHaltedForSafety {
			<-ctx.Done()
			return
		}

		// Otherwise, reset the synchronization state, but propagate the error
		// that caused failure.
		c.stateLock.Lock()
		c.state = &State{
			Session:    c.session,
			LastError:  err.Error(),
			AlphaState: &EndpointState{},
			BetaState:  &EndpointState{},
		}
		c.stateLock.Unlock()

		// If we were cancelled, then return immediately.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// If less than one auto-reconnect interval has elapsed since the last
		// synchronization failure, then wait before attempting reconnection.
		now := time.Now()
		if now.Sub(lastSynchronizationFailureTime) < autoReconnectInterval {
			select {
			case <-ctx.Done():
				return
			case <-time.After(autoReconnectInterval):
			}
		}
		lastSynchronizationFailureTime = now
	}
}

// synchronize is the main synchronization loop for the controller.
func (c *controller) synchronize(ctx context.Context, alpha, beta Endpoint) error {
	// Clear any error state upon restart of this function. If there was a
	// terminal error previously caused synchronization to fail, then the user
	// will have had time to review it (while the run loop is waiting to
	// reconnect), so it's not like we're getting rid of it too quickly.
	c.stateLock.Lock()
	if c.state.LastError != "" {
		c.state.LastError = ""
		c.stateLock.Unlock()
	} else {
		c.stateLock.UnlockWithoutNotify()
	}

	// Track whether or not a flush request triggered the synchronization loop.
	var flushRequest chan error

	// Load the archive and extract the ancestor. We enforce that the archive
	// contains only synchronizable content.
	archive := &core.Archive{}
	if err := encoding.LoadAndUnmarshalProtobuf(c.archivePath, archive); err != nil {
		return fmt.Errorf("unable to load archive: %w", err)
	} else if err = archive.EnsureValid(true); err != nil {
		return fmt.Errorf("invalid archive found on disk: %w", err)
	}
	ancestor := archive.Content

	// Compute the effective synchronization mode.
	synchronizationMode := c.session.Configuration.SynchronizationMode
	if synchronizationMode.IsDefault() {
		synchronizationMode = c.session.Version.DefaultSynchronizationMode()
	}

	// Compute the effective ignore syntax.
	ignoreSyntax := c.session.Configuration.IgnoreSyntax
	if ignoreSyntax.IsDefault() {
		ignoreSyntax = c.session.Version.DefaultIgnoreSyntax()
	}

	// Compute the effective permissions mode.
	permissionsMode := c.session.Configuration.PermissionsMode
	if permissionsMode.IsDefault() {
		permissionsMode = c.session.Version.DefaultPermissionsMode()
	}

	// Compute, on a per-endpoint basis, whether or not polling should be
	// disabled.
	αWatchMode := c.mergedAlphaConfiguration.WatchMode
	βWatchMode := c.mergedBetaConfiguration.WatchMode
	if αWatchMode.IsDefault() {
		αWatchMode = c.session.Version.DefaultWatchMode()
	}
	if βWatchMode.IsDefault() {
		βWatchMode = c.session.Version.DefaultWatchMode()
	}
	αDisablePolling := (αWatchMode == WatchMode_WatchModeNoWatch)
	βDisablePolling := (βWatchMode == WatchMode_WatchModeNoWatch)

	// Create a switch that will allow us to skip polling and force a
	// synchronization cycle. On startup, we enable this switch and skip polling
	// to immediately force a check for changes that may have occurred while the
	// synchronization loop wasn't running. The only time we don't force this
	// check on startup is when both endpoints have polling disabled, which is
	// an indication that the session should operate in a fully manual mode.
	skipPolling := (!αDisablePolling || !βDisablePolling)

	// Create variables to track our reasons for skipping polling.
	var skippingPollingDueToScanError, skippingPollingDueToMissingFiles bool

	// Loop until there is a synchronization error.
	for {
		// Unless we've been requested to skip polling, wait for a dirty state
		// while monitoring for cancellation. If we've been requested to skip
		// polling, it should only be for one iteration.
		if !skipPolling {
			// Update status to watching.
			c.stateLock.Lock()
			c.state.Status = Status_Watching
			c.stateLock.Unlock()

			// Create a polling context that we can cancel. We don't make it a
			// subcontext of our own cancellation context because it's easier to
			// just track cancellation there separately.
			pollCtx, pollCancel := context.WithCancel(context.Background())

			// Start alpha polling. If alpha has been put into a no-watch mode,
			// then we still perform polling in order to detect transport errors
			// that might occur while the session is sitting idle, but we ignore
			// any non-error responses and instead wait for the polling context
			// to be cancelled. We perform this ignore operation because we
			// don't want a broken or malicious endpoint to be able to force
			// synchronization, especially if its watching has been
			// intentionally disabled.
			//
			// It's worth noting that, because a well-behaved endpoint in
			// no-watch mode never returns events, we'll always be polling on it
			// (and thereby testing the transport) right up until the polling
			// context is cancelled. Thus, there's no need to worry about cases
			// where the endpoint sends back an event that we ignore and then
			// has a transport failure without us noticing while we wait on the
			// polling context (at least not for well-behaved endpoints).
			αPollResults := make(chan error, 1)
			go func() {
				if αDisablePolling {
					if err := alpha.Poll(pollCtx); err != nil {
						αPollResults <- err
					} else {
						<-pollCtx.Done()
						αPollResults <- nil
					}
				} else {
					αPollResults <- alpha.Poll(pollCtx)
				}
			}()

			// Start beta polling. The logic here mirrors that for alpha above.
			βPollResults := make(chan error, 1)
			go func() {
				if βDisablePolling {
					if err := beta.Poll(pollCtx); err != nil {
						βPollResults <- err
					} else {
						<-pollCtx.Done()
						βPollResults <- nil
					}
				} else {
					βPollResults <- beta.Poll(pollCtx)
				}
			}()

			// Wait for either poll to return an event or an error, for a flush
			// request, or for cancellation. In any of these cases, cancel
			// polling and ensure that both polling operations have completed.
			var αPollErr, βPollErr error
			cancelled := false
			select {
			case αPollErr = <-αPollResults:
				c.logger.Debug("Triggered by alpha endpoint")
				pollCancel()
				βPollErr = <-βPollResults
			case βPollErr = <-βPollResults:
				c.logger.Debug("Triggered by beta endpoint")
				pollCancel()
				αPollErr = <-αPollResults
			case flushRequest = <-c.flushRequests:
				if cap(flushRequest) < 1 {
					panic("unbuffered flush request")
				}
				c.logger.Debug("Triggered by flush request")
				pollCancel()
				αPollErr = <-αPollResults
				βPollErr = <-βPollResults
			case <-ctx.Done():
				cancelled = true
				pollCancel()
				αPollErr = <-αPollResults
				βPollErr = <-βPollResults
			}

			// Watch for errors or cancellation.
			if cancelled {
				return errors.New("cancelled during polling")
			} else if αPollErr != nil {
				return fmt.Errorf("alpha polling error: %w", αPollErr)
			} else if βPollErr != nil {
				return fmt.Errorf("beta polling error: %w", βPollErr)
			}
		} else {
			c.logger.Debug("Skipping polling")
			skipPolling = false
		}

		// Scan both endpoints in parallel and check for errors. If a flush
		// request is present, then force both endpoints to perform a full
		// (warm) re-scan rather than using acceleration.
		c.logger.Debug("Scanning endpoints")
		c.stateLock.Lock()
		c.state.Status = Status_Scanning
		c.stateLock.Unlock()
		forceFullScan := flushRequest != nil
		var αSnapshot, βSnapshot *core.Snapshot
		var αScanErr, βScanErr error
		var αTryAgain, βTryAgain bool
		scanDone := &sync.WaitGroup{}
		scanDone.Add(2)
		go func() {
			αSnapshot, αScanErr, αTryAgain = alpha.Scan(ctx, ancestor, forceFullScan)
			scanDone.Done()
		}()
		go func() {
			βSnapshot, βScanErr, βTryAgain = beta.Scan(ctx, ancestor, forceFullScan)
			scanDone.Done()
		}()
		scanDone.Wait()

		// Check if cancellation occurred during scanning.
		select {
		case <-ctx.Done():
			return errors.New("cancelled during scanning")
		default:
		}

		// Check for scan errors.
		if αScanErr != nil {
			αScanErr = fmt.Errorf("alpha scan error: %w", αScanErr)
			if !αTryAgain {
				return αScanErr
			} else {
				c.stateLock.Lock()
				c.state.LastError = αScanErr.Error()
				c.stateLock.Unlock()
			}
		}
		if βScanErr != nil {
			βScanErr = fmt.Errorf("beta scan error: %w", βScanErr)
			if !βTryAgain {
				return βScanErr
			} else {
				c.stateLock.Lock()
				c.state.LastError = βScanErr.Error()
				c.stateLock.Unlock()
			}
		}

		// Watch for retry recommendations from scan operations. These occur
		// when a scan fails and concurrent modifications are suspected as the
		// culprit. In these cases, we force another synchronization cycle. Note
		// that, because we skip polling, our flush request, if any, will still
		// be valid, and we'll be able to respond to it once a successful
		// synchronization cycle completes.
		//
		// TODO: Should we eventually abort synchronization after a certain
		// number of consecutive scan retries?
		if αTryAgain || βTryAgain {
			// If we're already in a synchronization cycle that was forced due
			// to a previous scan error, and we've now received another retry
			// recommendation, then wait before attempting a rescan.
			if skippingPollingDueToScanError {
				// Update status to waiting for rescan.
				c.stateLock.Lock()
				c.state.Status = Status_WaitingForRescan
				c.stateLock.Unlock()

				// Wait before trying to rescan, but watch for cancellation.
				select {
				case <-time.After(rescanWaitDuration):
				case <-ctx.Done():
					return errors.New("cancelled during rescan wait")
				}
			}

			// Retry.
			skipPolling = true
			skippingPollingDueToScanError = true
			continue
		}
		skippingPollingDueToScanError = false

		// Extract contents.
		αContent := αSnapshot.Content
		βContent := βSnapshot.Content
		if c.logger.Level() >= logging.LevelTrace {
			c.logger.Tracef("Ancestor contains %d entries, alpha contains %d entries, beta contains %d entries",
				ancestor.Count(), αContent.Count(), βContent.Count(),
			)
		}

		// If we're using Docker-style ignore syntax and semantics, then
		// snapshots may include phantom directories. In this case, we need to
		// perform a pre-processing step to reify these directories to either
		// tracked or ignored.
		αDirectoryCount := αSnapshot.Directories
		βDirectoryCount := βSnapshot.Directories
		if ignoreSyntax == ignore.Syntax_SyntaxDocker {
			αContent, βContent, αDirectoryCount, βDirectoryCount = core.ReifyPhantomDirectories(
				ancestor, αContent, βContent,
			)
		}

		// Now that we've had a successful scan, clear the last error (if any),
		// record scan statistics and problems (if any), and update the status
		// to reconciling.
		//
		// We know that it's okay to clear the error here (if there is one)
		// because we know that it originated from scan (since all other errors
		// are terminal and any previous terminal error would have been cleared
		// at the start of this function).
		c.stateLock.Lock()
		c.state.LastError = ""
		c.state.AlphaState.Scanned = true
		c.state.AlphaState.Directories = αDirectoryCount
		c.state.AlphaState.Files = αSnapshot.Files
		c.state.AlphaState.SymbolicLinks = αSnapshot.SymbolicLinks
		c.state.AlphaState.TotalFileSize = αSnapshot.TotalFileSize
		c.state.AlphaState.ScanProblems = αContent.Problems()
		c.state.BetaState.Scanned = true
		c.state.BetaState.Directories = βDirectoryCount
		c.state.BetaState.Files = βSnapshot.Files
		c.state.BetaState.SymbolicLinks = βSnapshot.SymbolicLinks
		c.state.BetaState.TotalFileSize = βSnapshot.TotalFileSize
		c.state.BetaState.ScanProblems = βContent.Problems()
		c.state.Status = Status_Reconciling
		c.stateLock.Unlock()

		// If we're propagating executability bits and one endpoint preserves
		// executability information while the the other does not, then
		// propagate executability information from the preserving side to the
		// non-preserving side. We only do this if the corresponding target
		// content is non-nil, because (a) PropagateExecutability is a no-op if
		// it is nil and (b) PreservesExecutability will have defaulted to false
		// if there's no content and (even though this will be a no-op) we don't
		// want the spurious logs.
		if permissionsMode == core.PermissionsMode_PermissionsModePortable {
			if αSnapshot.PreservesExecutability && βContent != nil && !βSnapshot.PreservesExecutability {
				c.logger.Debug("Propagating alpha executability to beta")
				βContent = core.PropagateExecutability(ancestor, αContent, βContent)
			} else if βSnapshot.PreservesExecutability && αContent != nil && !αSnapshot.PreservesExecutability {
				c.logger.Debug("Propagating beta executability to alpha")
				αContent = core.PropagateExecutability(ancestor, βContent, αContent)
			}
		}

		// Check if the root is a directory that's been emptied (by deleting a
		// non-trivial amount of content) on one endpoint (but not both). This
		// can be intentional, but usually indicates that a non-persistent
		// filesystem (such as a container filesystem) is being used as the
		// synchronization root. In any case, we switch to a halted state and
		// wait for the user to either manually propagate the deletion and
		// resume the session, recreate the session, or reset the session.
		if oneEndpointEmptiedRoot(ancestor, αContent, βContent) {
			c.stateLock.Lock()
			c.state.Status = Status_HaltedOnRootEmptied
			c.stateLock.Unlock()
			return errHaltedForSafety
		}

		// Perform reconciliation.
		c.logger.Debug("Performing reconciliation")
		ancestorChanges, αTransitions, βTransitions, conflicts := core.Reconcile(
			ancestor,
			αContent,
			βContent,
			synchronizationMode,
		)
		if c.logger.Level() >= logging.LevelTrace {
			for _, change := range ancestorChanges {
				c.logger.Tracef("Ancestor change at \"%s\" to %s",
					formatPathForLogging(change.Path),
					formatEntryForLogging(change.New),
				)
			}
			for _, transition := range αTransitions {
				c.logger.Tracef("Alpha transition at \"%s\" from %s to %s",
					formatPathForLogging(transition.Path),
					formatEntryForLogging(transition.Old),
					formatEntryForLogging(transition.New),
				)
			}
			for _, transition := range βTransitions {
				c.logger.Tracef("Beta transition at \"%s\" from %s to %s",
					formatPathForLogging(transition.Path),
					formatEntryForLogging(transition.Old),
					formatEntryForLogging(transition.New),
				)
			}
			for _, conflict := range conflicts {
				c.logger.Tracef("Conflict rooted at \"%s\"",
					formatPathForLogging(conflict.Root),
				)
			}
		}

		// Store conflicts that arose during reconciliation.
		c.stateLock.Lock()
		c.state.Conflicts = conflicts
		c.stateLock.Unlock()

		// Check if a root deletion operation is being propagated. This can be
		// intentional, accidental, or an indication of a non-persistent
		// filesystem (such as a container filesystem). In any case, we switch
		// to a halted state and wait for the user to either manually propagate
		// the deletion and resume the session, recreate the session, or reset
		// the session.
		if containsRootDeletion(αTransitions) || containsRootDeletion(βTransitions) {
			c.stateLock.Lock()
			c.state.Status = Status_HaltedOnRootDeletion
			c.stateLock.Unlock()
			return errHaltedForSafety
		}

		// Check if a root type change is being propagated. This can be
		// intentional or accidental. In any case, we switch to a halted state
		// and wait for the user to manually delete the content that will be
		// overwritten by the type change and resume the session.
		if containsRootTypeChange(αTransitions) || containsRootTypeChange(βTransitions) {
			c.stateLock.Lock()
			c.state.Status = Status_HaltedOnRootTypeChange
			c.stateLock.Unlock()
			return errHaltedForSafety
		}

		// Stage files on alpha.
		c.stateLock.Lock()
		c.state.Status = Status_StagingAlpha
		c.stateLock.Unlock()
		if paths, digests := core.TransitionDependencies(αTransitions); len(paths) > 0 {
			c.logger.Debugf("Staging %d file(s) on alpha", len(paths))
			filteredPaths, signatures, receiver, err := alpha.Stage(paths, digests)
			if err != nil {
				return fmt.Errorf("unable to begin staging on alpha: %w", err)
			}
			if !filteredPathsAreSubset(filteredPaths, paths) {
				return errors.New("alpha returned incorrect subset of staging paths")
			}
			if len(filteredPaths) < len(paths) {
				c.logger.Debugf("Alpha pre-staged %d/%d files", len(paths)-len(filteredPaths), len(paths))
			}
			if len(filteredPaths) > 0 {
				monitor := func(state *rsync.ReceiverState) error {
					c.stateLock.Lock()
					if state == nil {
						c.state.AlphaState.StagingProgress = nil
					} else {
						if c.state.AlphaState.StagingProgress == nil {
							c.state.AlphaState.StagingProgress = &rsync.ReceiverState{}
						}
						proto.Merge(c.state.AlphaState.StagingProgress, state)
					}
					c.stateLock.Unlock()
					return nil
				}
				receiver = rsync.NewMonitoringReceiver(receiver, filteredPaths, signatures, monitor)
				receiver = rsync.NewPreemptableReceiver(ctx, receiver)
				if err = beta.Supply(filteredPaths, signatures, receiver); err != nil {
					return fmt.Errorf("unable to stage files on alpha: %w", err)
				}
			}
		}

		// Stage files on beta.
		c.stateLock.Lock()
		c.state.Status = Status_StagingBeta
		c.stateLock.Unlock()
		if paths, digests := core.TransitionDependencies(βTransitions); len(paths) > 0 {
			c.logger.Debugf("Staging %d file(s) on beta", len(paths))
			filteredPaths, signatures, receiver, err := beta.Stage(paths, digests)
			if err != nil {
				return fmt.Errorf("unable to begin staging on beta: %w", err)
			}
			if !filteredPathsAreSubset(filteredPaths, paths) {
				return errors.New("beta returned incorrect subset of staging paths")
			}
			if len(filteredPaths) < len(paths) {
				c.logger.Debugf("Beta pre-staged %d/%d files", len(paths)-len(filteredPaths), len(paths))
			}
			if len(filteredPaths) > 0 {
				monitor := func(state *rsync.ReceiverState) error {
					c.stateLock.Lock()
					if state == nil {
						c.state.BetaState.StagingProgress = nil
					} else {
						if c.state.BetaState.StagingProgress == nil {
							c.state.BetaState.StagingProgress = &rsync.ReceiverState{}
						}
						proto.Merge(c.state.BetaState.StagingProgress, state)
					}
					c.stateLock.Unlock()
					return nil
				}
				receiver = rsync.NewMonitoringReceiver(receiver, filteredPaths, signatures, monitor)
				receiver = rsync.NewPreemptableReceiver(ctx, receiver)
				if err = alpha.Supply(filteredPaths, signatures, receiver); err != nil {
					return fmt.Errorf("unable to stage files on beta: %w", err)
				}
			}
		}

		// Perform transitions on both endpoints in parallel. For each side that
		// doesn't completely error out, convert its results to ancestor
		// changes. Transition errors are checked later, once the ancestor has
		// been updated.
		c.stateLock.Lock()
		c.state.Status = Status_Transitioning
		c.stateLock.Unlock()
		var αResults, βResults []*core.Entry
		var αProblems, βProblems []*core.Problem
		var αMissingFiles, βMissingFiles bool
		var αTransitionErr, βTransitionErr error
		var αChanges, βChanges []*core.Change
		transitionDone := &sync.WaitGroup{}
		if len(αTransitions) > 0 {
			transitionDone.Add(1)
		}
		if len(βTransitions) > 0 {
			transitionDone.Add(1)
		}
		if len(αTransitions) > 0 {
			c.logger.Debug("Transitioning alpha")
			go func() {
				αResults, αProblems, αMissingFiles, αTransitionErr = alpha.Transition(ctx, αTransitions)
				if αTransitionErr == nil {
					for t, transition := range αTransitions {
						αChanges = append(αChanges, &core.Change{Path: transition.Path, New: αResults[t]})
					}
				}
				transitionDone.Done()
			}()
		}
		if len(βTransitions) > 0 {
			c.logger.Debug("Transitioning beta")
			go func() {
				βResults, βProblems, βMissingFiles, βTransitionErr = beta.Transition(ctx, βTransitions)
				if βTransitionErr == nil {
					for t, transition := range βTransitions {
						βChanges = append(βChanges, &core.Change{Path: transition.Path, New: βResults[t]})
					}
				}
				transitionDone.Done()
			}()
		}
		transitionDone.Wait()

		// Record transition problems.
		c.stateLock.Lock()
		c.state.Status = Status_Saving
		c.state.AlphaState.TransitionProblems = αProblems
		c.state.BetaState.TransitionProblems = βProblems
		c.stateLock.Unlock()

		// Fold applied changes into the ancestor's change list and update the
		// ancestor if any changes are present.
		ancestorChanges = append(ancestorChanges, αChanges...)
		ancestorChanges = append(ancestorChanges, βChanges...)
		if len(ancestorChanges) > 0 {
			// Apply the changes to the ancestor.
			if newAncestor, err := core.Apply(ancestor, ancestorChanges); err != nil {
				return fmt.Errorf("unable to propagate changes to ancestor: %w", err)
			} else {
				ancestor = newAncestor
			}

			// Validate the new ancestor before saving it to ensure that our
			// reconciliation logic doesn't have any flaws. This is the only time
			// that we validate a data structure generated by code in the same
			// process (usually our tests are our validation), but this case is
			// special because (a) our test cases can't cover every real world
			// condition that might arise and (b) if we write a broken ancestor to
			// disk, the session is toast. This safety check ensures that even if we
			// put out a broken release, or encounter some bizarre real world merge
			// case that we didn't consider, things can be fixed.
			if err := ancestor.EnsureValid(true); err != nil {
				return fmt.Errorf("new ancestor is invalid: %w", err)
			}

			// Save the ancestor.
			c.logger.Debug("Saving ancestor")
			archive.Content = ancestor
			if err := encoding.MarshalAndSaveProtobuf(c.archivePath, archive); err != nil {
				return fmt.Errorf("unable to save ancestor: %w", err)
			}
		}

		// Now check for transition errors.
		if αTransitionErr != nil {
			return fmt.Errorf("unable to apply changes to alpha: %w", αTransitionErr)
		} else if βTransitionErr != nil {
			return fmt.Errorf("unable to apply changes to beta: %w", βTransitionErr)
		}

		// If there were files missing from either endpoint's stager during the
		// transition operations, then there were likely concurrent
		// modifications during staging. If we see this, then skip polling and
		// attempt to run another synchronization cycle immediately, but only if
		// we're not already in a synchronization cycle that was forced due to
		// previously missing files.
		if (αMissingFiles || βMissingFiles) && !skippingPollingDueToMissingFiles {
			c.logger.Debug("Endpoint(s) missing files after transition, skipping polling")
			skipPolling = true
			skippingPollingDueToMissingFiles = true
		} else {
			skippingPollingDueToMissingFiles = false
		}

		// Increment the synchronization cycle count.
		c.stateLock.Lock()
		c.state.SuccessfulCycles++
		c.stateLock.Unlock()

		// If a flush request triggered this synchronization cycle, then tell it
		// that the cycle has completed and remove it from our tracking.
		if flushRequest != nil {
			flushRequest <- nil
			flushRequest = nil
		}
	}
}
