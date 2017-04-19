package session

import (
	contextpkg "context"
	"io"
	"os"
	syncpkg "sync"
	"time"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/encoding"
	"github.com/havoc-io/mutagen/message"
	"github.com/havoc-io/mutagen/multiplex"
	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/state"
	"github.com/havoc-io/mutagen/sync"
	"github.com/havoc-io/mutagen/url"
)

const (
	autoReconnectInterval = 30 * time.Second
	rescanWaitDuration    = 5 * time.Second
)

type controller struct {
	// sessionPath is the path to the serialized session.
	sessionPath string
	// archivePath is the path to the serialized archive.
	archivePath string
	// stateLock guards and tracks changes to the session member's Paused field
	// and the state member. Code may access static members of the session
	// without holding this lock, but any reads or writes to the Paused field
	// (including as part of a read of the whole session) should be guarded by
	// this lock.
	stateLock *state.TrackingLock
	// session is the current session state. It should be saved to disk any time
	// it is modified.
	session *Session
	// state represents the current synchronization state.
	state SynchronizationState
	// lifecycleLock guards the disabled, cancel, and done members.
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
	// done will be closed by the current synchronization loop when it exits.
	done chan struct{}
}

func newSession(
	tracker *state.Tracker,
	alpha, beta *url.URL,
	ignores []string,
	prompter string,
) (*controller, error) {
	// TODO: Should we perform URL validation in here? They should be validated
	// by the respective dialers.

	// Verify that the ignores are valid.
	for _, ignore := range ignores {
		if !sync.ValidIgnorePattern(ignore) {
			return nil, errors.Errorf("invalid ignore specified: %s", ignore)
		}
	}

	// Attempt to connect. Session creation is only allowed after if successful.
	alphaConnection, err := connect(alpha, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to alpha")
	}
	betaConnection, err := connect(beta, prompter)
	if err != nil {
		alphaConnection.Close()
		return nil, errors.Wrap(err, "unable to connect to beta")
	}

	// Create the session and archive.
	session := &Session{
		Identifier:           uuid.NewV4().String(),
		Version:              Version_Version1,
		CreationTime:         time.Now(),
		CreatingVersionMajor: mutagen.VersionMajor,
		CreatingVersionMinor: mutagen.VersionMinor,
		CreatingVersionPatch: mutagen.VersionPatch,
		Alpha:                alpha,
		Beta:                 beta,
		Ignores:              ignores,
	}
	archive := &Archive{}

	// Compute session and archive paths.
	sessionPath, err := pathForSession(session.Identifier)
	if err != nil {
		alphaConnection.Close()
		betaConnection.Close()
		return nil, errors.Wrap(err, "unable to compute session path")
	}
	archivePath, err := pathForArchive(session.Identifier)
	if err != nil {
		alphaConnection.Close()
		betaConnection.Close()
		return nil, errors.Wrap(err, "unable to compute archive path")
	}

	// Save components to disk.
	if err := encoding.MarshalAndSaveProtobuf(sessionPath, session); err != nil {
		alphaConnection.Close()
		betaConnection.Close()
		return nil, errors.Wrap(err, "unable to save session")
	}
	if err := encoding.MarshalAndSaveProtobuf(archivePath, archive); err != nil {
		os.Remove(sessionPath)
		alphaConnection.Close()
		betaConnection.Close()
		return nil, errors.Wrap(err, "unable to save archive")
	}

	// Create the controller.
	controller := &controller{
		sessionPath: sessionPath,
		archivePath: archivePath,
		stateLock:   state.NewTrackingLock(tracker),
		session:     session,
	}

	// Start a synchronization loop.
	context, cancel := contextpkg.WithCancel(contextpkg.Background())
	controller.cancel = cancel
	controller.done = make(chan struct{})
	go controller.run(context, alphaConnection, betaConnection)

	// Success.
	return controller, nil
}

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

	// Load the session.
	session := &Session{}
	if err := encoding.LoadAndUnmarshalProtobuf(sessionPath, session); err != nil {
		return nil, errors.Wrap(err, "unable to load session configuration")
	}

	// Create the controller.
	controller := &controller{
		sessionPath: sessionPath,
		archivePath: archivePath,
		stateLock:   state.NewTrackingLock(tracker),
		session:     session,
	}

	// If the session isn't marked as paused, start a synchronization loop.
	if !session.Paused {
		context, cancel := contextpkg.WithCancel(contextpkg.Background())
		controller.cancel = cancel
		controller.done = make(chan struct{})
		go controller.run(context, nil, nil)
	}

	// Success.
	return controller, nil
}

func (c *controller) currentState() SessionState {
	// Lock the session state and defer its release. It's very important that we
	// unlock without a notification here, otherwise we'd trigger an infinite
	// cycle of list/notify.
	c.stateLock.Lock()
	defer c.stateLock.UnlockWithoutNotify()

	// Create the result. We make shallow copies of both state components. Both
	// technically have fields that contain mutable values, but these values are
	// treated as immutable so it is okay.
	result := SessionState{
		Session: &Session{},
		State:   c.state,
	}
	*result.Session = *c.session

	// Done.
	return result
}

func (c *controller) resume(prompter string) error {
	// Lock the controller's lifecycle and defer its release.
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Don't allow any resume operations if the controller is disabled.
	if c.disabled {
		return errors.New("controller disabled")
	}

	// Check if there's an existing synchronization loop.
	if c.cancel != nil {
		// If there is an existing synchronization loop, check if it's alredy
		// connected.
		c.stateLock.Lock()
		connected := c.state.Status > SynchronizationStatusConnecting
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
		c.done = nil
	}

	// Mark the session as unpaused and save it to disk.
	c.stateLock.Lock()
	c.session.Paused = false
	saveErr := encoding.MarshalAndSaveProtobuf(c.sessionPath, c.session)
	c.stateLock.Unlock()

	// Attempt to connect. This may fail for one or both of the endpoints, but
	// in that case we'll simply leave the session unpaused and allow it to try
	// to auto-reconnect later.
	alphaConnection, alphaConnectErr := connect(c.session.Alpha, prompter)
	betaConnection, betaConnectErr := connect(c.session.Beta, prompter)

	// Start the synchronization loop with what we have.
	context, cancel := contextpkg.WithCancel(contextpkg.Background())
	c.cancel = cancel
	c.done = make(chan struct{})
	go c.run(context, alphaConnection, betaConnection)

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

type haltMode uint8

const (
	haltModePause haltMode = iota
	haltModeShutdown
	haltModeTerminate
)

func (c *controller) halt(mode haltMode) error {
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

func (c *controller) run(context contextpkg.Context, alpha, beta io.ReadWriteCloser) {
	// Defer resource and state cleanup.
	defer func() {
		// Close any open connections. These might be open if the runloop was
		// cancelled while partially connected rather than after sync failure.
		if alpha != nil {
			alpha.Close()
		}
		if beta != nil {
			beta.Close()
		}

		// Reset the state.
		c.stateLock.Lock()
		c.state = SynchronizationState{}
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
		c.stateLock.Lock()
		c.state.Status = SynchronizationStatusConnecting
		c.stateLock.Unlock()
		for {
			// Ensure that alpha is connected.
			if alpha == nil {
				alpha, _ = reconnect(context, c.session.Alpha)
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
				beta, _ = reconnect(context, c.session.Beta)
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

		// Multiplex endpoint connections.
		alphaStreams, alphaMux := multiplex.ReadWriter(alpha, numberOfEndpointChannels)
		betaStreams, betaMux := multiplex.ReadWriter(beta, numberOfEndpointChannels)

		// Create a wait group that we can use to verify that all of our
		// background Goroutines have exited. This does not include the
		// synchronization Goroutine, which is monitored separately. We have 8
		// background Goroutines total (2 watch event tracking Goroutines, 2
		// rsync update tracking Goroutines, and 4 rsync forwarding Goroutines).
		var backgroundGoroutinesDone syncpkg.WaitGroup
		backgroundGoroutinesDone.Add(8)

		// Extract the event streams from each endpoint and convert them to
		// messaging streams. Create a channel that can be used to track dirty
		// states across both endpoints. Start two Goroutines
		dirty := make(chan watchEvent, 1)
		receiveWatchEventErrors := make(chan error, 2)
		go func() {
			receiveWatchEventErrors <- c.receiveWatchEvents(
				alphaStreams[endpointChannelWatchEvents],
				dirty,
			)
			backgroundGoroutinesDone.Done()
		}()
		go func() {
			receiveWatchEventErrors <- c.receiveWatchEvents(
				betaStreams[endpointChannelWatchEvents],
				dirty,
			)
			backgroundGoroutinesDone.Done()
		}()

		// Extract the update channels for each endpoint and convert them to
		// messaging streams. Start listening for updates in the background and
		// monitor for failure.
		receiveRsyncUpdateErrors := make(chan error, 2)
		go func() {
			receiveRsyncUpdateErrors <- c.receiveRsyncUpdates(
				alphaStreams[endpointChannelRsyncUpdates],
				true,
			)
			backgroundGoroutinesDone.Done()
		}()
		go func() {
			receiveRsyncUpdateErrors <- c.receiveRsyncUpdates(
				betaStreams[endpointChannelRsyncUpdates],
				false,
			)
			backgroundGoroutinesDone.Done()
		}()

		// Set up rsync forwarding between endpoints and monitor for failure.
		forwardingErrors := make(chan error, 4)
		go func() {
			// Foward the alpha client to the beta server.
			_, err := io.Copy(
				betaStreams[endpointChannelRsyncServer],
				alphaStreams[endpointChannelRsyncClient],
			)
			forwardingErrors <- err
			backgroundGoroutinesDone.Done()
		}()
		go func() {
			// Foward the beta client to the alpha server.
			_, err := io.Copy(
				alphaStreams[endpointChannelRsyncServer],
				betaStreams[endpointChannelRsyncClient],
			)
			forwardingErrors <- err
			backgroundGoroutinesDone.Done()
		}()
		go func() {
			// Forward the beta server to the alpha client.
			_, err := io.Copy(
				alphaStreams[endpointChannelRsyncClient],
				betaStreams[endpointChannelRsyncServer],
			)
			forwardingErrors <- err
			backgroundGoroutinesDone.Done()
		}()
		go func() {
			// Forward the alpha server to the beta client.
			_, err := io.Copy(
				betaStreams[endpointChannelRsyncClient],
				alphaStreams[endpointChannelRsyncServer],
			)
			forwardingErrors <- err
			backgroundGoroutinesDone.Done()
		}()

		// Create a cancellable sub-context for synchronization. We need this so
		// that we can stop synchronization in the event of an error in one of
		// the background Goroutines.
		syncContext, syncCancel := contextpkg.WithCancel(context)

		// Synchronize with these endpoints in a separate Goroutine.
		synchronizeErrors := make(chan error, 1)
		go func() {
			synchronizeErrors <- c.synchronize(
				syncContext,
				alphaStreams[endpointChannelControl],
				betaStreams[endpointChannelControl],
				dirty,
			)
		}()

		// Wait for any component to fail. We don't monitor for cancellation
		// explicitly in here because synchronize will see the cancellation and
		// return.
		var failureCause error
		synchronizeDone := false
		select {
		case err := <-receiveWatchEventErrors:
			failureCause = errors.Wrap(err, "watch event receiving error")
		case err := <-receiveRsyncUpdateErrors:
			failureCause = errors.Wrap(err, "rsync update receiving error")
		case err := <-forwardingErrors:
			failureCause = errors.Wrap(err, "rsync forwarding error")
		case err := <-synchronizeErrors:
			failureCause = errors.Wrap(err, "synchronization error")
			synchronizeDone = true
		}

		// In case it wasn't synchronization that failed, cancel it.
		syncCancel()

		// Ensure that the synchronization Goroutine has exited before closing
		// the multiplexers and underlying connections. We wait because we might
		// be in the middle of a transition and it's possible that the
		// connection has failed in a way that wouldn't prevent completion.
		if !synchronizeDone {
			<-synchronizeErrors
		}

		// Close the multiplexers.
		alphaMux.Close()
		betaMux.Close()

		// Close the underlying endpoint connections.
		alpha.Close()
		alpha = nil
		beta.Close()
		beta = nil

		// Wait until all Goroutines have exited before resetting state. We have
		// to do this because some Goroutines set state concurrently.
		backgroundGoroutinesDone.Wait()

		// Reset the synchronization state, but propagate the error that caused
		// failure.
		c.stateLock.Lock()
		c.state = SynchronizationState{
			LastError: failureCause.Error(),
		}
		c.stateLock.Unlock()

		// Do a non-blocking check for cancellation so that we don't waste
		// resources by trying another connect when the context has been
		// cancelled (it'll be wasteful).ontrol is better than sentinel errors.
		select {
		case <-context.Done():
			return
		default:
		}
	}
}

func (c *controller) receiveWatchEvents(connection io.ReadWriter, dirty chan watchEvent) error {
	// Convert the connection to a message stream.
	events := message.NewMessageStream(connection)

	// Receive watch events until there's an error.
	for {
		// Receive the next watch event.
		var event watchEvent
		if err := events.Decode(&event); err != nil {
			return errors.Wrap(err, "unable to receive watch event")
		}

		// Forward it in a non-blocking manner.
		select {
		case dirty <- event:
		default:
		}
	}
}

func (c *controller) receiveRsyncUpdates(connection io.ReadWriter, alpha bool) error {
	// Convert the connection to a message stream.
	updates := message.NewMessageStream(connection)

	// Receive updates until there's an error.
	for {
		// Receive the next status update.
		var status rsync.StagingStatus
		if err := updates.Decode(&status); err != nil {
			return errors.Wrap(err, "unable to receive rsync status update")
		}

		// Update the state.
		c.stateLock.Lock()
		if alpha {
			c.state.AlphaStaging = status
		} else {
			c.state.BetaStaging = status
		}
		c.stateLock.Unlock()
	}
}

func (c *controller) synchronize(
	context contextpkg.Context,
	alphaConnection, betaConnection io.ReadWriter,
	dirty chan watchEvent,
) error {
	// Update status to initializing.
	c.stateLock.Lock()
	c.state.Status = SynchronizationStatusInitializing
	c.stateLock.Unlock()

	// Convert the connections to message streams.
	alpha := message.NewMessageStream(alphaConnection)
	beta := message.NewMessageStream(betaConnection)

	// Load the archive and extract the ancestor.
	archive := &Archive{}
	if err := encoding.LoadAndUnmarshalProtobuf(c.archivePath, archive); err != nil {
		return errors.Wrap(err, "unable to load archive")
	}
	ancestor := archive.Root

	// Perform initialization on each of the endpoints.
	alphaPreservesExecutability, err := c.initialize(alpha, true)
	if err != nil {
		return errors.Wrap(err, "unable to initialize alpha")
	}
	betaPreservesExecutability, err := c.initialize(beta, false)
	if err != nil {
		return errors.Wrap(err, "unable to initialize beta")
	}

	// Loop until there is a synchronization error. We always skip polling on
	// the first time through the loop because changes may have occurred while
	// we were halted.
	skipPolling := true
	for {
		// Set status to scanning.
		c.stateLock.Lock()
		c.state.Status = SynchronizationStatusScanning
		c.stateLock.Unlock()

		// Unless we've been requested to skip polling, wait for a dirty state
		// while monitoring for cancellation. If we've been requested to skip
		// polling, it should only be for one iteration.
		if !skipPolling {
			select {
			case <-dirty:
			case <-context.Done():
				return errors.New("cancelled")
			}
		} else {
			skipPolling = false
		}

		// Perform scans.
		alphaSnapshot, alphaTryAgain, alphaScanErr := c.scan(
			alpha,
			alphaPreservesExecutability,
			ancestor,
		)
		if alphaScanErr != nil {
			return errors.Wrap(err, "alpha scan error")
		} else if alphaTryAgain {
			skipPolling = true
			continue
		}
		betaSnapshot, betaTryAgain, betaScanErr := c.scan(
			beta,
			betaPreservesExecutability,
			ancestor,
		)
		if betaScanErr != nil {
			return errors.Wrap(err, "beta scan error")
		} else if betaTryAgain {
			skipPolling = true
			continue
		}

		// Update status to reconciling.
		c.stateLock.Lock()
		c.state.Status = SynchronizationStatusReconciling
		c.stateLock.Unlock()

		// Reconcile.
		ancestorChanges, alphaTransitions, betaTransitions, conflicts := sync.Reconcile(
			ancestor, alphaSnapshot, betaSnapshot,
		)

		// Update status to staging and record conflicts.
		c.stateLock.Lock()
		c.state.Status = SynchronizationStatusStaging
		c.state.Conflicts = conflicts
		c.stateLock.Unlock()

		// Create staging result channels.
		alphaStagingErrors := make(chan error, 1)
		betaStagingErrors := make(chan error, 1)

		// Start staging in separate Goroutines.
		go func() {
			alphaStagingErrors <- c.stage(alpha, alphaTransitions)
		}()
		go func() {
			betaStagingErrors <- c.stage(beta, betaTransitions)
		}()

		// Wait for both stagings to complete. Because staging can take a long
		// time to complete and is safe to interrupt, we poll for cancellation
		// here. Once we return, the underlying connection will be closed and
		// the staging Goroutines will error out.
		var alphaStagingError, betaStagingError error
		var alphaStagingDone, betaStagingDone bool
		for !alphaStagingDone || !betaStagingDone {
			select {
			case alphaStagingError = <-alphaStagingErrors:
				alphaStagingDone = true
			case betaStagingError = <-betaStagingErrors:
				betaStagingDone = true
			case <-context.Done():
				return errors.New("cancelled")
			}
		}

		// Check for staging errors.
		if alphaStagingError != nil {
			return errors.Wrap(alphaStagingError, "alpha staging error")
		} else if betaStagingError != nil {
			return errors.Wrap(betaStagingError, "beta staging error")
		}

		// Update status to transitioning.
		c.stateLock.Lock()
		c.state.Status = SynchronizationStatusTransitioning
		c.stateLock.Unlock()

		// Perform transitions. We don't allow this to be cancelled by the
		// synchroniztion context because we might lose information on changes
		// that we've made. This is fine, because this method won't block
		// indefinitely and should be relatively fast. We don't abort
		// immediately on error, because we want to propagate any changes that
		// we make.
		alphaChanges, alphaProblems, alphaTransitionErr := c.transition(alpha, alphaTransitions)
		betaChanges, betaProblems, betaTransitionErr := c.transition(beta, betaTransitions)

		// Record problems.
		c.stateLock.Lock()
		c.state.AlphaProblems = alphaProblems
		c.state.BetaProblems = betaProblems
		c.stateLock.Unlock()

		// Update status to transitioning.
		c.stateLock.Lock()
		c.state.Status = SynchronizationStatusSaving
		c.stateLock.Unlock()

		// Combine changes and propagate them to the ancestor. Even if there
		// were transition errors, this code is still valid.
		ancestorChanges = append(ancestorChanges, alphaChanges...)
		ancestorChanges = append(ancestorChanges, betaChanges...)
		ancestor, err = sync.Apply(ancestor, ancestorChanges)
		if err != nil {
			return errors.Wrap(err, "unable to propagate changes to ancestor")
		}

		// Validate the new ancestor before saving it to ensure that our
		// reconciliation logic doesn't have any flaws.
		if err = ancestor.EnsureValid(); err != nil {
			return errors.Wrap(err, "new ancestor is invalid")
		}

		// Save the ancestor.
		archive.Root = ancestor
		if err = encoding.MarshalAndSaveProtobuf(c.archivePath, archive); err != nil {
			return errors.Wrap(err, "unable to save ancestor")
		}

		// Now check for transition errors.
		if alphaTransitionErr != nil {
			return errors.Wrap(err, "unable to apply changes to alpha")
		} else if betaTransitionErr != nil {
			return errors.Wrap(err, "unable to apply changes to beta")
		}

		// After a successful synchronization cycle, clear any synchronization
		// error.
		c.stateLock.Lock()
		c.state.LastError = ""
		c.stateLock.Unlock()
	}
}

func (c *controller) initialize(endpoint message.MessageStream, alpha bool) (bool, error) {
	// Create the initialization request.
	root := c.session.Alpha.Path
	if !alpha {
		root = c.session.Beta.Path
	}
	request := initializeRequest{
		Session: c.session.Identifier,
		Version: c.session.Version,
		Root:    root,
		Ignores: c.session.Ignores,
		Alpha:   alpha,
	}

	// Send the request.
	if err := endpoint.Encode(request); err != nil {
		return false, errors.Wrap(err, "unable to send initialize request")
	}

	// Receive the response.
	var response initializeResponse
	if err := endpoint.Decode(&response); err != nil {
		return false, errors.Wrap(err, "unable to receive initialize response")
	}

	// Success.
	return response.PreservesExecutability, nil
}

func (c *controller) scan(
	endpoint message.MessageStream,
	preservesExecutability bool,
	ancestor *sync.Entry,
) (*sync.Entry, bool, error) {
	// Create an rsync engine.
	rsyncer := rsync.NewEngine()

	// Start by expecting the ancestor as a base.
	expected := ancestor

	// If the endpoint doesn't preserve executability, then strip executability
	// bits from the expected snapshot since the incoming value won't have them.
	if !preservesExecutability {
		expected = sync.StripExecutability(expected)
	}

	// Marshal the expected snapshot.
	expectedBytes, err := marshalEntry(expected)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to marshal expected snapshot")
	}

	// Compute the base snapshot signature.
	expectedSignature := rsyncer.BytesSignature(
		expectedBytes,
		rsync.OptimalBlockSizeForBaseLength(uint64(len(expectedBytes))),
	)

	// Send the request.
	request := endpointRequest{Scan: &scanRequest{expectedSignature}}
	if err := endpoint.Encode(request); err != nil {
		return nil, false, errors.Wrap(err, "unable to send scan request")
	}

	// Read the response.
	var response scanResponse
	if err := endpoint.Decode(&response); err != nil {
		return nil, false, errors.Wrap(err, "unable to receive scan response")
	}

	// Check if the endpoint says we should try again.
	if response.TryAgain {
		return nil, true, nil
	}

	// Apply the remote's deltas to the expected snapshot.
	snapshotBytes, err := rsyncer.PatchBytes(expectedBytes, expectedSignature, response.SnapshotDelta)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to patch base snapshot")
	}

	// Unmarshal the snapshot.
	snapshot, err := unmarshalEntry(snapshotBytes)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to unmarshal snapshot")
	}

	// If the endpoint doesn't preserve executability, then propagate
	// executability from the ancestor.
	if !preservesExecutability {
		snapshot = sync.PropagateExecutability(ancestor, snapshot)
	}

	// Success.
	return snapshot, false, nil
}

func (c *controller) stage(endpoint message.MessageStream, transitions []sync.Change) error {
	// Send the request.
	request := endpointRequest{Stage: &stageRequest{transitions}}
	if err := endpoint.Encode(request); err != nil {
		return errors.Wrap(err, "unable to send staging request")
	}

	// Receive the response.
	var response stageResponse
	if err := endpoint.Decode(&response); err != nil {
		return errors.Wrap(err, "unable to receive staging response")
	}

	// Success.
	return nil
}

func (c *controller) transition(endpoint message.MessageStream, transitions []sync.Change) ([]sync.Change, []sync.Problem, error) {
	// Send the request.
	request := endpointRequest{Transition: &transitionRequest{transitions}}
	if err := endpoint.Encode(request); err != nil {
		return nil, nil, errors.Wrap(err, "unable to send transition request")
	}

	// Receive the response.
	var response transitionResponse
	if err := endpoint.Decode(&response); err != nil {
		return nil, nil, errors.Wrap(err, "unable to receive initialize response")
	}

	// Success.
	return response.Changes, response.Problems, nil
}
