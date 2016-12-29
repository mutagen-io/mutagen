package session

import (
	contextpkg "context"
	"io"
	"os"
	syncpkg "sync"
	"time"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes"

	uuid "github.com/satori/go.uuid"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/encoding"
	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/state"
	"github.com/havoc-io/mutagen/stream"
	"github.com/havoc-io/mutagen/sync"
	"github.com/havoc-io/mutagen/url"
)

const (
	autoReconnectInterval = 30 * time.Second
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
	defaultIgnores bool, ignores []string,
	prompter string,
) (*controller, error) {
	// TODO: Should we perform URL validation in here? They should be validated
	// by the respective dialers.

	// Verify that the ignores are valid.
	for _, ignore := range ignores {
		if !sync.ValidIgnore(ignore) {
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
	creationTime, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		alphaConnection.Close()
		betaConnection.Close()
		return nil, errors.Wrap(err, "unable to compute session creation time")
	}
	session := &Session{
		Identifier:           uuid.NewV4().String(),
		Version:              Version_Version1,
		CreationTime:         creationTime,
		CreatingVersionMajor: mutagen.VersionMajor,
		CreatingVersionMinor: mutagen.VersionMinor,
		CreatingVersionPatch: mutagen.VersionPatch,
		Alpha:                alpha,
		Beta:                 beta,
		DefaultIgnores:       defaultIgnores,
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
		// enforce this invariant as part of this type's logic).
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

	// Attempt to connect.
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
	// Register cleanup.
	defer func() {
		// Close any open connections.
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
				if α, err := reconnect(context, c.session.Alpha); err != nil {
					select {
					case <-context.Done():
						return
					default:
					}
				} else {
					alpha = α
				}
			}
			c.stateLock.Lock()
			c.state.AlphaConnected = (alpha != nil)
			c.stateLock.Unlock()

			// Ensure that beta is connected.
			if beta == nil {
				if β, err := reconnect(context, c.session.Beta); err != nil {
					select {
					case <-context.Done():
						return
					default:
					}
				} else {
					beta = β
				}
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

		// Multiplex connections to the endpoint. Also nil-out the corresponding
		// connections since they're owned by the multiplexers now.
		alphaMultiplexer := multiplex(alpha, false)
		betaMultiplexer := multiplex(beta, false)
		alpha, beta = nil, nil

		// Forward connections between endpoints and monitor for forwarding
		// failure. These Goroutines and any they spawn will die when the
		// multiplexers die.
		alphaMultiplexerForwardingErrors := make(chan error, 1)
		betaMultiplexerForwardingErrors := make(chan error, 1)
		go func() {
			alphaMultiplexerForwardingErrors <- errors.Wrap(stream.Forward(
				alphaMultiplexer, betaMultiplexer,
			), "alpha connection forwarding failure")
		}()
		go func() {
			betaMultiplexerForwardingErrors <- errors.Wrap(stream.Forward(
				betaMultiplexer, alphaMultiplexer,
			), "beta connection forwarding failure")
		}()

		// Create RPC clients for each endpoint.
		alphaClient := rpc.NewClient(alphaMultiplexer)
		betaClient := rpc.NewClient(betaMultiplexer)

		// Create a cancellable sub-context for synchronization. We need this so
		// that we can stop synchronization in the event of forwarding errors.
		syncContext, syncCancel := contextpkg.WithCancel(context)

		// Synchronize with these endpoints in a separate Goroutine.
		synchronizeErrors := make(chan error, 1)
		go func() {
			synchronizeErrors <- errors.Wrap(c.synchronize(
				syncContext, alphaClient, betaClient,
			), "synchronization failure")
		}()

		// Wait for forwarding or synchronization to fail. We don't monitor for
		// cancellation explicitly in here because synchronize will see the
		// cancellation and return.
		var synchronizeErr error
		synchronizeExited := false
		select {
		case synchronizeErr = <-alphaMultiplexerForwardingErrors:
		case synchronizeErr = <-betaMultiplexerForwardingErrors:
		case synchronizeErr = <-synchronizeErrors:
			synchronizeExited = true
		}

		// In case it was one of the forwarders that failed, cancel the
		// synchronization context.
		syncCancel()

		// Ensure that the synchronization loop has exited.
		if !synchronizeExited {
			<-synchronizeErrors
		}

		// Close both multiplexers.
		alphaMultiplexer.Close()
		betaMultiplexer.Close()

		// Reset the synchronization state, but propagate the error that caused
		// failure.
		c.stateLock.Lock()
		c.state = SynchronizationState{
			LastError: synchronizeErr.Error(),
		}
		c.stateLock.Unlock()

		// Do a non-blocking check for cancellation so that we don't waste
		// resources by trying another connect when the context has been
		// cancelled (it'll be wasteful). This is better than sentinel errors.
		select {
		case <-context.Done():
			return
		default:
		}
	}
}

func (c *controller) synchronize(context contextpkg.Context, alpha, beta *rpc.Client) error {
	// Update status to initializing.
	c.stateLock.Lock()
	c.state.Status = SynchronizationStatusInitializing
	c.stateLock.Unlock()

	// Load the archive.
	archive := &Archive{}
	if err := encoding.LoadAndUnmarshalProtobuf(c.archivePath, archive); err != nil {
		// TODO: This seems like it should be a fairly terminal error, but I'm
		// not sure how to signal that to the run loop.
		return errors.Wrap(err, "unable to load archive")
	}

	// Set up the initial state.
	ancestor := archive.Root
	alphaExpected := ancestor
	betaExpected := ancestor

	// Perform initialization on each of the endpoints.
	alphaPreservesExecutability, err := c.initialize(context, alpha, true)
	if err != nil {
		return errors.Wrap(err, "unable to initialize alpha")
	}
	betaPreservesExecutability, err := c.initialize(context, beta, false)
	if err != nil {
		return errors.Wrap(err, "unable to initialize beta")
	}

	// Loop until there is a synchronization error.
	for {
		// Set status to scanning.
		c.stateLock.Lock()
		c.state.Status = SynchronizationStatusScanning
		c.stateLock.Unlock()

		// Create a context that will allow us to force a scan to complete.
		unforced, force := contextpkg.WithCancel(contextpkg.Background())

		// Create scan result channels.
		alphaScanResults := make(chan scanResult, 1)
		betaScanResults := make(chan scanResult, 1)

		// Start scanning in separate Goroutines.
		go c.scan(context, alpha, unforced, alphaPreservesExecutability, ancestor, alphaExpected, alphaScanResults)
		go c.scan(context, beta, unforced, betaPreservesExecutability, ancestor, betaExpected, betaScanResults)

		// Wait for the first scan to complete. When that happens, regardless of
		// whether or not it was successful, force the other scan to complete
		// immediately and wait for its result.
		var alphaResult, betaResult scanResult
		var alphaDone bool
		select {
		case alphaResult = <-alphaScanResults:
			alphaDone = true
		case betaResult = <-betaScanResults:
		}
		force()
		if alphaDone {
			betaResult = <-betaScanResults
		} else {
			alphaResult = <-alphaScanResults
		}

		// Check for errors.
		if alphaResult.error != nil {
			return errors.Wrap(alphaResult.error, "alpha scan failed")
		} else if betaResult.error != nil {
			return errors.Wrap(betaResult.error, "beta scan failed")
		}

		// Extract results.
		alphaExpected = alphaResult.snapshot
		betaExpected = betaResult.snapshot

		// Update status to reconciling.
		c.stateLock.Lock()
		c.state.Status = SynchronizationStatusReconciling
		c.stateLock.Unlock()

		// Reconcile.
		ancestorChanges, alphaTransitions, betaTransitions, conflicts := sync.Reconcile(
			ancestor, alphaExpected, betaExpected,
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
			alphaStagingErrors <- c.stage(context, alpha, true, alphaTransitions)
		}()
		go func() {
			betaStagingErrors <- c.stage(context, beta, false, betaTransitions)
		}()

		// Wait for both stagings to complete.
		var alphaStagingError, betaStagingError error
		var alphaStagingDone, betaStagingDone bool
		for !alphaStagingDone || !betaStagingDone {
			select {
			case alphaStagingError = <-alphaStagingErrors:
				alphaStagingDone = true
			case betaStagingError = <-betaStagingErrors:
				betaStagingDone = true
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
		if err := ancestor.EnsureValid(); err != nil {
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

func (c *controller) initialize(context contextpkg.Context, endpoint *rpc.Client, alpha bool) (bool, error) {
	// Invoke the method.
	stream, err := endpoint.Invoke(endpointMethodInitialize)
	if err != nil {
		return false, errors.Wrap(err, "unable to invoke initialization")
	}

	// Ensure that the stream is closed either on context cancellation or when
	// we return.
	cancellableContext, contextCancel := contextpkg.WithCancel(context)
	go func() {
		<-cancellableContext.Done()
		stream.Close()
	}()
	defer contextCancel()

	// Create the initialize request.
	root := c.session.Alpha.Path
	if !alpha {
		root = c.session.Beta.Path
	}
	request := initializeRequest{
		Session:        c.session.Identifier,
		Version:        c.session.Version,
		Root:           root,
		DefaultIgnores: c.session.DefaultIgnores,
		Ignores:        c.session.Ignores,
		Alpha:          alpha,
	}

	// Send the request.
	if err := stream.Send(request); err != nil {
		return false, errors.Wrap(err, "unable to send initialize request")
	}

	// Receive the response.
	var response initializeResponse
	if err := stream.Receive(&response); err != nil {
		return false, errors.Wrap(err, "unable to receive initialize response")
	}

	// Success.
	return response.PreservesExecutability, nil
}

type scanResult struct {
	snapshot *sync.Entry
	error    error
}

func (c *controller) scan(
	context contextpkg.Context,
	endpoint *rpc.Client,
	unforced contextpkg.Context,
	preservesExecutability bool,
	ancestor, expected *sync.Entry,
	results chan scanResult,
) {
	// If the capacity of the results channel is less than one, this is a logic
	// error.
	if cap(results) < 1 {
		panic("scan provided with non-buffered channel")
	}

	// Create a function that can send an error result.
	sendError := func(err error) {
		results <- scanResult{error: err}
	}

	// If the endpoint doesn't preserve executability, then strip executability
	// bits from the expected snapshot since the incoming value won't have them.
	if !preservesExecutability {
		expected = sync.StripExecutability(expected)
	}

	// Marshal the expected snapshot into a stable format.
	expectedBytes, err := stableMarshal(expected)
	if err != nil {
		sendError(errors.Wrap(err, "unable to marshal expected snapshot"))
		return
	}

	// Create an rsyncer.
	rsyncer := rsync.New()

	// Compute the expected snapshot signature.
	expectedSignature := rsyncer.BytesSignature(expectedBytes)

	// Compute the expected snapshot checksum.
	expectedChecksum := snapshotChecksum(expectedBytes)

	// Invoke the scan.
	stream, err := endpoint.Invoke(endpointMethodScan)
	if err != nil {
		sendError(errors.Wrap(err, "unable to invoke scan"))
		return
	}

	// Ensure that the stream is closed either on context cancellation or when
	// we return.
	cancellableContext, contextCancel := contextpkg.WithCancel(context)
	go func() {
		<-cancellableContext.Done()
		stream.Close()
	}()
	defer contextCancel()

	// Send the request.
	if err := stream.Send(scanRequest{
		BaseSnapshotSignature:    expectedSignature,
		ExpectedSnapshotChecksum: expectedChecksum,
	}); err != nil {
		sendError(errors.Wrap(err, "unable to send scan request"))
		return
	}

	// Start a Goroutine that will send a force request if the unforced context
	// is cancelled. We create a context around the unforced context whose
	// cancellation we defer to ensure that this Goroutine always exits when
	// we're done. We don't watch for errors in sending the force request
	// because they will only be transport errors, either due to the stream
	// being closed (in which case we're done) or an actual error (in which case
	// we'll see it when trying to receive the response below).
	cancellableUnforced, force := contextpkg.WithCancel(unforced)
	go func() {
		<-cancellableUnforced.Done()
		stream.Send(scanRequest{})
	}()
	defer force()

	// Read the response.
	var response scanResponse
	if err := stream.Receive(&response); err != nil {
		sendError(errors.Wrap(err, "unable to receive scan response"))
		return
	}

	// Apply the remote's deltas to the expected snapshot.
	snapshotBytes, err := rsyncer.PatchBytes(expectedBytes, response.SnapshotDelta, nil)
	if err != nil {
		sendError(errors.Wrap(err, "unable to patch base snapshot"))
		return
	}

	// Unmarshal the snapshot.
	snapshot, err := stableUnmarshal(snapshotBytes)
	if err != nil {
		sendError(errors.Wrap(err, "unable to unmarshal snapshot"))
		return
	}

	// If the endpoint doesn't preserve executability, then propagate
	// executability from the ancestor.
	if !preservesExecutability {
		snapshot = sync.PropagateExecutability(ancestor, snapshot)
	}

	// Success.
	results <- scanResult{snapshot: snapshot}
}

func (c *controller) stage(
	context contextpkg.Context,
	endpoint *rpc.Client,
	alpha bool,
	transitions []sync.Change,
) error {
	// Invoke the method.
	stream, err := endpoint.Invoke(endpointMethodStage)
	if err != nil {
		return errors.Wrap(err, "unable to invoke staging")
	}

	// Ensure that the stream is closed either on context cancellation or when
	// we return.
	cancellableContext, contextCancel := contextpkg.WithCancel(context)
	go func() {
		<-cancellableContext.Done()
		stream.Close()
	}()
	defer contextCancel()

	// Send the request.
	if err := stream.Send(stageRequest{Transitions: transitions}); err != nil {
		return errors.Wrap(err, "unable to send staging request")
	}

	// Receive responses and update state until there's an error or completion.
	for {
		var response stageResponse
		if err := stream.Receive(&response); err == io.EOF {
			c.stateLock.Lock()
			if alpha {
				c.state.AlphaStaging = StagingStatus{}
			} else {
				c.state.BetaStaging = StagingStatus{}
			}
			c.stateLock.Unlock()
			return nil
		} else if err != nil {
			// We don't clear the state if there's an error, because it's going
			// to be reset when the synchronization loop exits anyway.
			return errors.Wrap(err, "unable to receive staging response")
		} else {
			c.stateLock.Lock()
			if alpha {
				c.state.AlphaStaging = response.Status
			} else {
				c.state.BetaStaging = response.Status
			}
			c.stateLock.Unlock()
		}
	}
}

func (c *controller) transition(endpoint *rpc.Client, transitions []sync.Change) ([]sync.Change, []sync.Problem, error) {
	// Invoke the method. Ensure that the stream is closed when we're done. We
	// don't allow transition to be interrupted by a context.
	stream, err := endpoint.Invoke(endpointMethodTransition)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to invoke transition")
	}
	defer stream.Close()

	// Send the request.
	if err := stream.Send(&transitionRequest{Transitions: transitions}); err != nil {
		return nil, nil, errors.Wrap(err, "unable to send transition request")
	}

	// Receive the response.
	var response transitionResponse
	if err := stream.Receive(&response); err != nil {
		return nil, nil, errors.Wrap(err, "unable to receive initialize response")
	}

	// Success.
	return response.Changes, response.Problems, nil
}
