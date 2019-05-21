package session

import (
	contextpkg "context"
	"fmt"
	"os"
	syncpkg "sync"
	"time"

	"github.com/pkg/errors"

	"github.com/google/uuid"

	"github.com/golang/protobuf/ptypes"

	"github.com/havoc-io/mutagen/pkg/encoding"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/prompt"
	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/state"
	"github.com/havoc-io/mutagen/pkg/sync"
	"github.com/havoc-io/mutagen/pkg/url"
)

const (
	// autoReconnectInterval is the period of time to wait before attempting an
	// automatic reconnect after disconnection or a failed reconnect.
	autoReconnectInterval = 30 * time.Second
	// rescanWaitDuration is the period of time to wait before attempting to
	// rescan after an ephemeral scan failure.
	rescanWaitDuration = 5 * time.Second
)

// controller manages and executes a single session.
type controller struct {
	// sessionPath is the path to the serialized session.
	sessionPath string
	// archivePath is the path to the serialized archive.
	archivePath string
	// stateLock guards and tracks changes to the session member's Paused field
	// and the state member.
	stateLock *state.TrackingLock
	// session records the associated session metadata and configuration. It is
	// considered static and safe for concurrent access except for its Paused
	// field, for which the stateLock member should be held. It should be saved
	// to disk any time it is modified.
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
	// lifecycleLock guards setting of the disabled, cancel, flushRequests, and
	// done members. Access to these members is allowed for the synchronization
	// loop without holding the lock. Any code wishing to set these members
	// should first acquire the lock, then cancel the synchronization loop, and
	// wait for it to complete before making any such changes.
	lifecycleLock syncpkg.Mutex
	// disabled indicates that no more changes to the synchronization loop
	// lifecycle are allowed (i.e. no more synchronization loops can be started
	// for this controller). This is used by terminate and shutdown. It should
	// only be set to true once any existing synchronization loop has been
	// stopped.
	disabled bool
	// cancel cancels the synchronization loop execution context. It should be
	// nil if and only if there is no synchronization loop running.
	cancel contextpkg.CancelFunc
	// flushRequests is used pass flush requests to the synchronization loop. It
	// is buffered, allowing a single request to be queued. All requests passed
	// via that channel must be buffered and contain room for one error.
	flushRequests chan chan error
	// done will be closed by the current synchronization loop when it exits.
	done chan struct{}
}

// newSession creates a new session and corresponding controller.
func newSession(
	tracker *state.Tracker,
	alpha, beta *url.URL,
	configuration, configurationAlpha, configurationBeta *Configuration,
	labels map[string]string,
	prompter string,
) (*controller, error) {
	// Update status.
	prompt.Message(prompter, "Creating session...")

	// Create a unique session identifier.
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate UUID for session")
	}
	identifier := randomUUID.String()

	// Set the session version.
	version := Version_Version1

	// Compute the creation time and convert it to Protocol Buffers format.
	creationTime := time.Now()
	creationTimeProto, err := ptypes.TimestampProto(creationTime)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert creation time format")
	}

	// Compute merged endpoint configurations.
	mergedAlphaConfiguration := MergeConfigurations(configuration, configurationAlpha)
	mergedBetaConfiguration := MergeConfigurations(configuration, configurationBeta)

	// Attempt to connect. Session creation is only allowed after if successful.
	alphaEndpoint, err := connect(alpha, prompter, identifier, version, mergedAlphaConfiguration, true)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to alpha")
	}
	betaEndpoint, err := connect(beta, prompter, identifier, version, mergedBetaConfiguration, false)
	if err != nil {
		alphaEndpoint.Shutdown()
		return nil, errors.Wrap(err, "unable to connect to beta")
	}

	// Create the session and archive.
	session := &Session{
		Identifier:           identifier,
		Version:              version,
		CreationTime:         creationTimeProto,
		CreatingVersionMajor: mutagen.VersionMajor,
		CreatingVersionMinor: mutagen.VersionMinor,
		CreatingVersionPatch: mutagen.VersionPatch,
		Alpha:                alpha,
		Beta:                 beta,
		Configuration:        configuration,
		ConfigurationAlpha:   configurationAlpha,
		ConfigurationBeta:    configurationBeta,
		Labels:               labels,
	}
	archive := &sync.Archive{}

	// Compute session and archive paths.
	sessionPath, err := pathForSession(session.Identifier)
	if err != nil {
		alphaEndpoint.Shutdown()
		betaEndpoint.Shutdown()
		return nil, errors.Wrap(err, "unable to compute session path")
	}
	archivePath, err := pathForArchive(session.Identifier)
	if err != nil {
		alphaEndpoint.Shutdown()
		betaEndpoint.Shutdown()
		return nil, errors.Wrap(err, "unable to compute archive path")
	}

	// Save components to disk.
	if err := encoding.MarshalAndSaveProtobuf(sessionPath, session); err != nil {
		alphaEndpoint.Shutdown()
		betaEndpoint.Shutdown()
		return nil, errors.Wrap(err, "unable to save session")
	}
	if err := encoding.MarshalAndSaveProtobuf(archivePath, archive); err != nil {
		os.Remove(sessionPath)
		alphaEndpoint.Shutdown()
		betaEndpoint.Shutdown()
		return nil, errors.Wrap(err, "unable to save archive")
	}

	// Create the controller.
	controller := &controller{
		sessionPath:              sessionPath,
		archivePath:              archivePath,
		stateLock:                state.NewTrackingLock(tracker),
		session:                  session,
		mergedAlphaConfiguration: mergedAlphaConfiguration,
		mergedBetaConfiguration:  mergedBetaConfiguration,
		state: &State{
			Session: session,
		},
	}

	// Start a synchronization loop.
	context, cancel := contextpkg.WithCancel(contextpkg.Background())
	controller.cancel = cancel
	controller.flushRequests = make(chan chan error, 1)
	controller.done = make(chan struct{})
	go controller.run(context, alphaEndpoint, betaEndpoint)

	// Success.
	return controller, nil
}

// loadSession loads an existing session and creates a corresponding controller.
func loadSession(tracker *state.Tracker, identifier string) (*controller, error) {
	// Compute session and archive paths.
	sessionPath, err := pathForSession(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute session path")
	}
	archivePath, err := pathForArchive(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute archive path")
	}

	// Load and validate the session. We have to populate a few optional fields
	// before validation if they're not set. We can't do this in the Session
	// literal because they'll be wiped out during unmarshalling, even if not
	// set.
	session := &Session{}
	if err := encoding.LoadAndUnmarshalProtobuf(sessionPath, session); err != nil {
		return nil, errors.Wrap(err, "unable to load session configuration")
	}
	if session.ConfigurationAlpha == nil {
		session.ConfigurationAlpha = &Configuration{}
	}
	if session.ConfigurationBeta == nil {
		session.ConfigurationBeta = &Configuration{}
	}
	if err := session.EnsureValid(); err != nil {
		return nil, errors.Wrap(err, "invalid session found on disk")
	}

	// Create the controller.
	controller := &controller{
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
			Session: session,
		},
	}

	// If the session isn't marked as paused, start a synchronization loop.
	if !session.Paused {
		context, cancel := contextpkg.WithCancel(contextpkg.Background())
		controller.cancel = cancel
		controller.flushRequests = make(chan chan error, 1)
		controller.done = make(chan struct{})
		go controller.run(context, nil, nil)
	}

	// Success.
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
	return c.state.Copy()
}

// flush attempts to force a synchronization cycle for the session. If wait is
// specified, then the method will wait until a post-flush synchronization cycle
// has completed. The provided context (which must be non-nil) can terminate
// this wait early.
func (c *controller) flush(prompter string, skipWait bool, context contextpkg.Context) error {
	// Update status.
	prompt.Message(prompter, fmt.Sprintf("Forcing synchronization cycle for session %s...", c.session.Identifier))

	// Lock the controller's lifecycle and defer its release.
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Don't allow any operations if the controller is disabled.
	if c.disabled {
		return errors.New("controller disabled")
	}

	// Check if the session is paused by checking whether or not it has a
	// synchronization loop. It's an internal invariant that a session has a
	// synchronization loop if and only if it is not paused, but checking for
	// paused status requires holding the state lock, whereas checking for the
	// existence of a synchronization loop only requires holding the lifecycle
	// lock, which we do at this point.
	if c.cancel == nil {
		return errors.New("session is paused")
	}

	// Create a flush request.
	request := make(chan error, 1)

	// If we don't want to wait, then we can simply send the request in a
	// non-blocking manner, in which case either this request (or one that's
	// already queued) will be processed eventually. After that, we'd done.
	if skipWait {
		// Send the request in a non-blocking manner.
		select {
		case c.flushRequests <- request:
		default:
		}

		// Success.
		return nil
	}

	// Otherwise we need to send the request in a blocking manner, watching for
	// cancellation in the mean time.
	select {
	case c.flushRequests <- request:
	case <-context.Done():
		return errors.New("flush cancelled before request could be sent")
	}

	// Now we need to wait for a response to the request, again watching for
	// cancellation in the mean time.
	select {
	case err := <-request:
		if err != nil {
			return err
		}
	case <-context.Done():
		return errors.New("flush cancelled while waiting for synchronization cycle")
	}

	// Success.
	return nil
}

// resume attempts to reconnect and resume the session if it isn't currently
// connected and synchronizing.
func (c *controller) resume(prompter string) error {
	// Update status.
	prompt.Message(prompter, fmt.Sprintf("Resuming session %s...", c.session.Identifier))

	// Lock the controller's lifecycle and defer its release.
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Don't allow any resume operations if the controller is disabled.
	if c.disabled {
		return errors.New("controller disabled")
	}

	// Check if there's an existing synchronization loop (i.e. if the session is
	// unpaused).
	if c.cancel != nil {
		// If there is an existing synchronization loop, check if it's already
		// in a state that's considered "connected".
		c.stateLock.Lock()
		connected := c.state.Status >= Status_Watching
		c.stateLock.UnlockWithoutNotify()

		// If we're already connect, then there's nothing we need to do. We
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
		c.session.Alpha,
		prompter,
		c.session.Identifier,
		c.session.Version,
		c.mergedAlphaConfiguration,
		true,
	)
	c.stateLock.Lock()
	c.state.AlphaConnected = (alpha != nil)
	c.stateLock.Unlock()

	// Attempt to connect to beta.
	c.stateLock.Lock()
	c.state.Status = Status_ConnectingBeta
	c.stateLock.Unlock()
	beta, betaConnectErr := connect(
		c.session.Beta,
		prompter,
		c.session.Identifier,
		c.session.Version,
		c.mergedBetaConfiguration,
		false,
	)
	c.stateLock.Lock()
	c.state.BetaConnected = (beta != nil)
	c.stateLock.Unlock()

	// Start the synchronization loop with what we have. Alpha or beta may have
	// failed to connect (and be nil), but in any case that'll just make the run
	// loop keep trying to connect.
	context, cancel := contextpkg.WithCancel(contextpkg.Background())
	c.cancel = cancel
	c.flushRequests = make(chan chan error, 1)
	c.done = make(chan struct{})
	go c.run(context, alpha, beta)

	// Report any errors. Since we always want to start a synchronization loop,
	// even on partial or complete failure (since it might be able to
	// auto-reconnect on its own), we wait until the end to report errors.
	if saveErr != nil {
		return errors.Wrap(saveErr, "unable to save session configuration")
	} else if alphaConnectErr != nil {
		return errors.Wrap(alphaConnectErr, "unable to connect to alpha")
	} else if betaConnectErr != nil {
		return errors.Wrap(betaConnectErr, "unable to connect to beta")
	}

	// Success.
	return nil
}

// haltMode represents the behavior to use when halting a session.
type haltMode uint8

const (
	// haltModePause indicates that a session should be halted and marked as
	// paused.
	haltModePause haltMode = iota
	// haltModeShutdown indicates that a session should be halted.
	haltModeShutdown
	// haltModeShutdown indicates that a session should be halted and then
	// deleted.
	haltModeTerminate
)

// description returns a human-readable description of a halt mode.
func (m haltMode) description() string {
	switch m {
	case haltModePause:
		return "Pausing"
	case haltModeShutdown:
		return "Shutting down"
	case haltModeTerminate:
		return "Terminating"
	default:
		panic("unhandled halt mode")
	}
}

// halt halts the session with the specified behavior.
func (c *controller) halt(mode haltMode, prompter string) error {
	// Update status.
	prompt.Message(prompter, fmt.Sprintf("%s session %s...", mode.description(), c.session.Identifier))

	// Lock the controller's lifecycle and defer its release.
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Don't allow any additional halt operations if the controller is disabled,
	// because either this session is being terminated or the service is
	// shutting down, and in either case there is no point in halting.
	if c.disabled {
		return errors.New("controller disabled")
	}

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
	if mode == haltModePause {
		// Mark the session as paused and save it.
		c.stateLock.Lock()
		c.session.Paused = true
		err := encoding.MarshalAndSaveProtobuf(c.sessionPath, c.session)
		c.stateLock.Unlock()
		if err != nil {
			return errors.Wrap(err, "unable to save session state")
		}
	} else if mode == haltModeShutdown {
		// Disable the controller.
		c.disabled = true
	} else if mode == haltModeTerminate {
		// Disable the controller.
		c.disabled = true

		// Wipe the session information from disk.
		sessionRemoveErr := os.Remove(c.sessionPath)
		archiveRemoveErr := os.Remove(c.archivePath)
		if sessionRemoveErr != nil {
			return errors.Wrap(sessionRemoveErr, "unable to remove session from disk")
		} else if archiveRemoveErr != nil {
			return errors.Wrap(archiveRemoveErr, "unable to remove archive from disk")
		}
	} else {
		panic("invalid halt mode specified")
	}

	// Success.
	return nil
}

// run is the main runloop for the controller, managing connectivity and
// synchronization.
func (c *controller) run(context contextpkg.Context, alpha, beta Endpoint) {
	// Defer resource and state cleanup.
	defer func() {
		// Shutdown any endpoints. These might be non-nil if the runloop was
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
			Session: c.session,
		}
		c.stateLock.Unlock()

		// Signal completion.
		close(c.done)
	}()

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
				alpha, _ = reconnect(
					context,
					c.session.Alpha,
					c.session.Identifier,
					c.session.Version,
					c.mergedAlphaConfiguration,
					true,
				)
			}
			c.stateLock.Lock()
			c.state.AlphaConnected = (alpha != nil)
			c.stateLock.Unlock()

			// Do a non-blocking check for cancellation to avoid a spurious
			// connection to beta in case cancellation occurred while connecting
			// to alpha.
			select {
			case <-context.Done():
				return
			default:
			}

			// Ensure that beta is connected.
			if beta == nil {
				c.stateLock.Lock()
				c.state.Status = Status_ConnectingBeta
				c.stateLock.Unlock()
				beta, _ = reconnect(
					context,
					c.session.Beta,
					c.session.Identifier,
					c.session.Version,
					c.mergedBetaConfiguration,
					false,
				)
			}
			c.stateLock.Lock()
			c.state.BetaConnected = (beta != nil)
			c.stateLock.Unlock()

			// If both endpoints are connected, we're done. We perform this
			// check here (rather than in the loop condition) because if we did
			// it in the loop condition we'd still need this check to avoid a
			// sleep every time (even if already successfully connected).
			if alpha != nil && beta != nil {
				break
			}

			// If we failed to connect, wait and then retry. Watch for
			// cancellation in the mean time.
			select {
			case <-context.Done():
				return
			case <-time.After(autoReconnectInterval):
			}
		}

		// Perform synchronization.
		err := c.synchronize(context, alpha, beta)

		// Shutdown the endpoints.
		alpha.Shutdown()
		alpha = nil
		beta.Shutdown()
		beta = nil

		// Reset the synchronization state, but propagate the error that caused
		// failure.
		c.stateLock.Lock()
		c.state = &State{
			Session:   c.session,
			LastError: err.Error(),
		}
		c.stateLock.Unlock()

		// If synchronization failed, wait and then try to reconnect. Watch for
		// cancellation in the mean time. This cancellation check will also
		// catch cases where the synchronization loop has been cancelled.
		select {
		case <-context.Done():
			return
		case <-time.After(autoReconnectInterval):
		}
	}
}

// synchronize is the main synchronization loop for the controller.
func (c *controller) synchronize(context contextpkg.Context, alpha, beta Endpoint) error {
	// Clear any error state upon restart of this function. If there was a
	// terminal error previously caused synchronization to fail, then the user
	// will have had 30 seconds to review it (while the run loop is waiting to
	// reconnect), so it's not like we're getting rid of it too quickly.
	c.stateLock.Lock()
	if c.state.LastError != "" {
		c.state.LastError = ""
		c.stateLock.Unlock()
	} else {
		c.stateLock.UnlockWithoutNotify()
	}

	// Track any flush request that we've pulled from the queue but haven't
	// marked as complete. If we bail due to an error, then close out the
	// request.
	var flushRequest chan error
	defer func() {
		if flushRequest != nil {
			flushRequest <- errors.New("synchronization cycle failed")
			flushRequest = nil
		}
	}()

	// Load the archive and extract the ancestor.
	archive := &sync.Archive{}
	if err := encoding.LoadAndUnmarshalProtobuf(c.archivePath, archive); err != nil {
		return errors.Wrap(err, "unable to load archive")
	} else if err = archive.Root.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid archive found on disk")
	}
	ancestor := archive.Root

	// Compute the effective synchronization mode.
	synchronizationMode := c.session.Configuration.SynchronizationMode
	if synchronizationMode.IsDefault() {
		synchronizationMode = c.session.Version.DefaultSynchronizationMode()
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
			pollContext, pollCancel := contextpkg.WithCancel(contextpkg.Background())

			// Start alpha polling. If alpha has been put into a no-watch mode,
			// then don't actually start polling there, because we don't want a
			// malfunctioning (or malicious) endpoint to be able to force
			// synchronization when it's not supposed to be able to.
			αPollResults := make(chan error, 1)
			go func() {
				if αDisablePolling {
					<-pollContext.Done()
					αPollResults <- nil
				} else {
					αPollResults <- alpha.Poll(pollContext)
				}
			}()

			// Start beta polling. The logic here mirrors that for alpha above.
			βPollResults := make(chan error, 1)
			go func() {
				if βDisablePolling {
					<-pollContext.Done()
					βPollResults <- nil
				} else {
					βPollResults <- beta.Poll(pollContext)
				}
			}()

			// Wait for either poll to return an event or an error, for a flush
			// request, or for cancellation. In any of these cases, cancel
			// polling and ensure that both polling operations have completed.
			var αPollErr, βPollErr error
			cancelled := false
			select {
			case αPollErr = <-αPollResults:
				pollCancel()
				βPollErr = <-βPollResults
			case βPollErr = <-βPollResults:
				pollCancel()
				αPollErr = <-αPollResults
			case flushRequest = <-c.flushRequests:
				if cap(flushRequest) < 1 {
					panic("unbuffered flush request")
				}
				pollCancel()
				αPollErr = <-αPollResults
				βPollErr = <-βPollResults
			case <-context.Done():
				cancelled = true
				pollCancel()
				αPollErr = <-αPollResults
				βPollErr = <-βPollResults
			}

			// Watch for errors or cancellation.
			if cancelled {
				return errors.New("cancelled during polling")
			} else if αPollErr != nil {
				return errors.Wrap(αPollErr, "alpha polling error")
			} else if βPollErr != nil {
				return errors.Wrap(βPollErr, "beta polling error")
			}
		} else {
			skipPolling = false
		}

		// Scan both endpoints in parallel and check for errors. If a flush
		// request is present, then force both endpoints to perform a full
		// (warm) re-scan rather than using acceleration.
		c.stateLock.Lock()
		c.state.Status = Status_Scanning
		c.stateLock.Unlock()
		forceFullScan := flushRequest != nil
		var αSnapshot, βSnapshot *sync.Entry
		var αPreservesExecutability, βPreservesExecutability bool
		var αScanErr, βScanErr error
		var αTryAgain, βTryAgain bool
		scanDone := &syncpkg.WaitGroup{}
		scanDone.Add(2)
		go func() {
			αSnapshot, αPreservesExecutability, αScanErr, αTryAgain = alpha.Scan(
				ancestor,
				forceFullScan,
			)
			scanDone.Done()
		}()
		go func() {
			βSnapshot, βPreservesExecutability, βScanErr, βTryAgain = beta.Scan(
				ancestor,
				forceFullScan,
			)
			scanDone.Done()
		}()
		scanDone.Wait()
		if αScanErr != nil {
			αScanErr = errors.Wrap(αScanErr, "alpha scan error")
			if !αTryAgain {
				return αScanErr
			} else {
				c.stateLock.Lock()
				c.state.LastError = αScanErr.Error()
				c.stateLock.Unlock()
			}
		}
		if βScanErr != nil {
			βScanErr = errors.Wrap(βScanErr, "beta scan error")
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
				case <-context.Done():
					return errors.New("cancelled during rescan wait")
				}
			}

			// Retry.
			skipPolling = true
			skippingPollingDueToScanError = true
			continue
		}
		skippingPollingDueToScanError = false

		// Clear the last error (if any) after a successful scan. Since scan
		// errors are the only non-terminal errors, and since we know that we've
		// cleared any other terminal error at the entry to this loop, we know
		// that what we're covering up here can only be a scan error.
		c.stateLock.Lock()
		if c.state.LastError != "" {
			c.state.LastError = ""
			c.stateLock.Unlock()
		} else {
			c.stateLock.UnlockWithoutNotify()
		}

		// If one side preserves executability and the other does not, then
		// propagate executability from the preserving side to the
		// non-preserving side.
		if αPreservesExecutability && !βPreservesExecutability {
			βSnapshot = sync.PropagateExecutability(ancestor, αSnapshot, βSnapshot)
		} else if βPreservesExecutability && !αPreservesExecutability {
			αSnapshot = sync.PropagateExecutability(ancestor, βSnapshot, αSnapshot)
		}

		// Update status to reconciling.
		c.stateLock.Lock()
		c.state.Status = Status_Reconciling
		c.stateLock.Unlock()

		// Perform reconciliation.
		ancestorChanges, αTransitions, βTransitions, conflicts := sync.Reconcile(
			ancestor,
			αSnapshot,
			βSnapshot,
			synchronizationMode,
		)

		// Create a slim copy of the conflicts so that we don't need to hold
		// the full-size versions in memory or send them over the wire.
		var slimConflicts []*sync.Conflict
		if len(conflicts) > 0 {
			slimConflicts = make([]*sync.Conflict, len(conflicts))
			for c, conflict := range conflicts {
				slimConflicts[c] = conflict.CopySlim()
			}
		}
		c.stateLock.Lock()
		c.state.Conflicts = slimConflicts
		c.stateLock.Unlock()

		// Check if a root deletion is being propagated. If so, switch to a
		// halted state. This is a best-effort safety check. While we'll
		// definitely detect root deletion, it may happen (for directories) that
		// a large portion of the root is deleted before the root itself and
		// that this partial deletion is captured during a scan and propagated
		// and the root deletion is not detected until the next scan. Scans have
		// been designed to completely abort if they detect concurrent
		// modifications, and there's a high cross-section for this detection.
		// But there's no silver bullet for this problem, at least not without
		// rendering Mutagen basically useless when it comes to propagating
		// changes. We may be able to add a flag to avoid propagating deletions
		// of any kind in the future, but that would be specialized and a
		// relatively harsh restriction to put in place as a general safety
		// mechanism.
		rootDeletion := false
		for _, t := range αTransitions {
			if isRootDeletion(t) {
				rootDeletion = true
				break
			}
		}
		if !rootDeletion {
			for _, t := range βTransitions {
				if isRootDeletion(t) {
					rootDeletion = true
					break
				}
			}
		}
		if rootDeletion {
			c.stateLock.Lock()
			c.state.Status = Status_HaltedOnRootDeletion
			c.stateLock.Unlock()
			<-context.Done()
			return errors.New("cancelled while halted on root deletion")
		}

		// Check if a root type change is being propagated and halt if so.
		rootTypeChange := false
		for _, t := range αTransitions {
			if isRootTypeChange(t) {
				rootTypeChange = true
				break
			}
		}
		if !rootTypeChange {
			for _, t := range βTransitions {
				if isRootTypeChange(t) {
					rootTypeChange = true
					break
				}
			}
		}
		if rootTypeChange {
			c.stateLock.Lock()
			c.state.Status = Status_HaltedOnRootTypeChange
			c.stateLock.Unlock()
			<-context.Done()
			return errors.New("cancelled while halted on root type change")
		}

		// Create a monitoring callback for rsync staging.
		monitor := func(status *rsync.ReceiverStatus) error {
			c.stateLock.Lock()
			c.state.StagingStatus = status
			c.stateLock.Unlock()
			return nil
		}

		// Stage files on alpha.
		c.stateLock.Lock()
		c.state.Status = Status_StagingAlpha
		c.stateLock.Unlock()
		if paths, digests, err := sync.TransitionDependencies(αTransitions); err != nil {
			return errors.Wrap(err, "unable to determine paths for staging on alpha")
		} else if len(paths) > 0 {
			filteredPaths, signatures, receiver, err := alpha.Stage(paths, digests)
			if err != nil {
				return errors.Wrap(err, "unable to begin staging on alpha")
			}
			if !filteredPathsAreSubset(filteredPaths, paths) {
				return errors.New("alpha returned incorrect subset of staging paths")
			}
			if len(filteredPaths) > 0 {
				receiver = rsync.NewMonitoringReceiver(receiver, filteredPaths, monitor)
				receiver = rsync.NewPreemptableReceiver(receiver, context)
				if err = beta.Supply(filteredPaths, signatures, receiver); err != nil {
					return errors.Wrap(err, "unable to stage files on alpha")
				}
			}
		}

		// Stage files on beta.
		c.stateLock.Lock()
		c.state.Status = Status_StagingBeta
		c.stateLock.Unlock()
		if paths, digests, err := sync.TransitionDependencies(βTransitions); err != nil {
			return errors.Wrap(err, "unable to determine paths for staging on beta")
		} else if len(paths) > 0 {
			filteredPaths, signatures, receiver, err := beta.Stage(paths, digests)
			if err != nil {
				return errors.Wrap(err, "unable to begin staging on beta")
			}
			if !filteredPathsAreSubset(filteredPaths, paths) {
				return errors.New("beta returned incorrect subset of staging paths")
			}
			if len(filteredPaths) > 0 {
				receiver = rsync.NewMonitoringReceiver(receiver, filteredPaths, monitor)
				receiver = rsync.NewPreemptableReceiver(receiver, context)
				if err = alpha.Supply(filteredPaths, signatures, receiver); err != nil {
					return errors.Wrap(err, "unable to stage files on beta")
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
		var αResults, βResults []*sync.Entry
		var αProblems, βProblems []*sync.Problem
		var αMissingFiles, βMissingFiles bool
		var αTransitionErr, βTransitionErr error
		var αChanges, βChanges []*sync.Change
		transitionDone := &syncpkg.WaitGroup{}
		if len(αTransitions) > 0 {
			transitionDone.Add(1)
		}
		if len(βTransitions) > 0 {
			transitionDone.Add(1)
		}
		if len(αTransitions) > 0 {
			go func() {
				αResults, αProblems, αMissingFiles, αTransitionErr = alpha.Transition(αTransitions)
				if αTransitionErr == nil {
					for t, transition := range αTransitions {
						αChanges = append(αChanges, &sync.Change{Path: transition.Path, New: αResults[t]})
					}
				}
				transitionDone.Done()
			}()
		}
		if len(βTransitions) > 0 {
			go func() {
				βResults, βProblems, βMissingFiles, βTransitionErr = beta.Transition(βTransitions)
				if βTransitionErr == nil {
					for t, transition := range βTransitions {
						βChanges = append(βChanges, &sync.Change{Path: transition.Path, New: βResults[t]})
					}
				}
				transitionDone.Done()
			}()
		}
		transitionDone.Wait()

		// Record problems and then combine changes and propagate them to the
		// ancestor. Even if there were transition errors, this code is still
		// valid.
		c.stateLock.Lock()
		c.state.Status = Status_Saving
		c.state.AlphaProblems = αProblems
		c.state.BetaProblems = βProblems
		c.stateLock.Unlock()
		ancestorChanges = append(ancestorChanges, αChanges...)
		ancestorChanges = append(ancestorChanges, βChanges...)
		if newAncestor, err := sync.Apply(ancestor, ancestorChanges); err != nil {
			return errors.Wrap(err, "unable to propagate changes to ancestor")
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
		if err := ancestor.EnsureValid(); err != nil {
			return errors.Wrap(err, "new ancestor is invalid")
		}

		// Save the ancestor.
		archive.Root = ancestor
		if err := encoding.MarshalAndSaveProtobuf(c.archivePath, archive); err != nil {
			return errors.Wrap(err, "unable to save ancestor")
		}

		// Now check for transition errors.
		if αTransitionErr != nil {
			return errors.Wrap(αTransitionErr, "unable to apply changes to alpha")
		} else if βTransitionErr != nil {
			return errors.Wrap(βTransitionErr, "unable to apply changes to beta")
		}

		// If there were files missing from either endpoint's stager during the
		// transition operations, then there were likely concurrent
		// modifications during staging. If we see this, then skip polling and
		// attempt to run another synchronization cycle immediately, but only if
		// we're not already in a synchronization cycle that was forced due to
		// previously missing files.
		if (αMissingFiles || βMissingFiles) && !skippingPollingDueToMissingFiles {
			skipPolling = true
			skippingPollingDueToMissingFiles = true
		} else {
			skippingPollingDueToMissingFiles = false
		}

		// Increment the synchronization cycle count.
		c.stateLock.Lock()
		c.state.SuccessfulSynchronizationCycles++
		c.stateLock.Unlock()

		// If a flush request triggered this synchronization cycle, then tell it
		// that the cycle has completed and remove it from our tracking.
		if flushRequest != nil {
			flushRequest <- nil
			flushRequest = nil
		}
	}
}
