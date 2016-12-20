package session

import (
	contextpkg "context"
	"io"
	"os"
	syncpkg "sync"
	"time"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/encoding"
	"github.com/havoc-io/mutagen/rpc"
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
	// stateLock guards and tracks changes to the session and state members. It
	// also guards the on-disk session serialization.
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
	// TODO: Implement.
	return nil, errors.New("not implemented")
}

func loadSession(tracker *state.Tracker, identifier string) (*controller, error) {
	// TODO: Implement.
	return nil, errors.New("not implemented")
}

func (c *controller) resume(prompter string) error {
	// TODO: Implement.
	return errors.New("not implemented")
}

func (c *controller) currentState() SessionState {
	// TODO: Implement.
	return SessionState{}
}

type haltMode uint8

const (
	haltModePause haltMode = iota
	haltModeShutdown
	haltModeTerminate
)

func (c *controller) halt(mode haltMode) error {
	// Lock the controller and defer its release.
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Don't allow any additional halt operations if the controller is disabled,
	// because either this session is being terminated or the service is
	// shutting down.
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
		c.stateLock.Lock()
		sessionRemoveErr := os.Remove(c.sessionPath)
		archiveRemoveErr := os.Remove(c.archivePath)
		c.stateLock.UnlockWithoutNotify()
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
		if alpha == nil {
			alpha.Close()
		}
		if beta == nil {
			beta.Close()
		}

		// Reset the state.
		c.stateLock.Lock()
		c.state = SynchronizationState{}
		c.stateLock.Unlock()

		// Signal completion.
		close(c.done)
	}()

	// Grab URLs.
	c.stateLock.Lock()
	alphaURL := c.session.Alpha
	betaURL := c.session.Beta
	c.stateLock.UnlockWithoutNotify()

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
				if α, err := reconnect(context, alphaURL); err != nil {
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
				if β, err := reconnect(context, betaURL); err == nil {
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

		// Create a cancellable sub-context for synchronization.
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
			Error: synchronizeErr.Error(),
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
		c.state.Status = SynchronizationStatusInitializing
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

		// Perform application. We don't abort immediately on error, because we
		// want to propagate any changes that we make.
		alphaChanges, alphaProblems, alphaApplyErr := c.apply(alpha, alphaTransitions)
		betaChanges, betaProblems, betaApplyErr := c.apply(beta, betaTransitions)

		// Record problems.
		c.stateLock.Lock()
		c.state.AlphaProblems = alphaProblems
		c.state.BetaProblems = betaProblems
		c.stateLock.Unlock()

		// Combine changes and propagate them to the ancestor. Even if there
		// were apply errors, this code is still valid.
		ancestorChanges = append(ancestorChanges, alphaChanges...)
		ancestorChanges = append(ancestorChanges, betaChanges...)
		ancestor, err = sync.Apply(ancestor, ancestorChanges)
		if err != nil {
			return errors.Wrap(err, "unable to propagate changes to ancestor")
		}

		// Save the ancestor.
		archive.Root = ancestor
		if err = encoding.MarshalAndSaveProtobuf(c.archivePath, archive); err != nil {
			return errors.Wrap(err, "unable to save ancestor")
		}

		// Now check for apply errors.
		if alphaApplyErr != nil {
			return errors.Wrap(err, "unable to apply changes to alpha")
		} else if betaApplyErr != nil {
			return errors.Wrap(err, "unable to apply changes to beta")
		}
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
	c.stateLock.Lock()
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
	c.stateLock.UnlockWithoutNotify()

	// Send the request.
	if err := stream.Encode(request); err != nil {
		return false, errors.Wrap(err, "unable to send initialize request")
	}

	// Receive the response.
	var response initializeResponse
	if err := stream.Decode(&response); err != nil {
		return false, errors.Wrap(err, "unable to decode response")
	}

	// Check for initialization errors on the remote.
	if response.Error != "" {
		return false, errors.Wrap(errors.New(response.Error), "remote initialize error")
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

	// Marshal the expected snapshot.
	expectedBytes, err := expected.Encode()
	if err != nil {
		sendError(errors.Wrap(err, "unable to marshal expected snapshot"))
		return
	}

	// Compute the expected snapshot signature.
	expectedSignature, err := snapshotSignature(expectedBytes)
	if err != nil {
		sendError(errors.Wrap(err, "unable to compute expected snapshot signature"))
		return
	}

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
	if err := stream.Encode(scanRequest{
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
	// we'll see it when trying to decode the response below).
	cancellableUnforced, force := contextpkg.WithCancel(unforced)
	go func() {
		<-cancellableUnforced.Done()
		stream.Encode(scanRequest{})
	}()
	defer force()

	// Read the response.
	var response scanResponse
	if err := stream.Decode(&response); err != nil {
		sendError(errors.Wrap(err, "unable to receive scan response"))
		return
	}

	// Check for scan errors on the remote.
	if response.Error != "" {
		sendError(errors.Wrap(errors.New(response.Error), "remote scan error"))
		return
	}

	// Apply the remote's deltas to the expected snapshot.
	snapshotBytes, err := patchSnapshot(expectedBytes, response.SnapshotDelta)
	if err != nil {
		sendError(errors.Wrap(err, "unable to patch base snapshot"))
		return
	}

	// Unmarshal the snapshot.
	snapshot, err := sync.DecodeEntry(snapshotBytes)
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
	if err := stream.Encode(stageRequest{Transitions: transitions}); err != nil {
		return errors.Wrap(err, "unable to send staging request")
	}

	// Receive responses and update state until there's an error or completion.
	for {
		var response stageResponse
		if err := stream.Decode(&response); err != nil {
			return errors.Wrap(err, "unable to receive staging response")
		} else if response.Error != "" {
			return errors.Wrap(errors.New(response.Error), "remote staging error")
		} else if response.Done {
			c.stateLock.Lock()
			if alpha {
				c.state.AlphaStaging = StagingStatus{}
			} else {
				c.state.BetaStaging = StagingStatus{}
			}
			c.stateLock.Unlock()
			return nil
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

func (c *controller) apply(endpoint *rpc.Client, transitions []sync.Change) ([]sync.Change, []sync.Problem, error) {
	// Invoke the method. Ensure that the stream is closed when we're done. We
	// don't allow apply to be interrupted by a context.
	stream, err := endpoint.Invoke(endpointMethodApply)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to invoke application")
	}
	defer stream.Close()

	// Send the request.
	if err := stream.Encode(&applyRequest{Transitions: transitions}); err != nil {
		return nil, nil, errors.Wrap(err, "unable to send apply request")
	}

	// Receive the response.
	var response applyResponse
	if err := stream.Decode(&response); err != nil {
		return nil, nil, errors.Wrap(err, "unable to decode response")
	}

	// Check for apply errors on the remote.
	if response.Error != "" {
		return nil, nil, errors.Wrap(errors.New(response.Error), "remote apply error")
	}

	// Success.
	return response.Changes, response.Problems, nil
}
