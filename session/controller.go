package session

import (
	contextpkg "context"
	"os"
	syncpkg "sync"
	"time"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/encoding"
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

	// Create a session identifier.
	identifier := uuid.NewV4().String()

	// Set the session version.
	version := Version_Version1

	// Attempt to connect. Session creation is only allowed after if successful.
	alphaEndpoint, err := connect(identifier, version, alpha, ignores, true, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to alpha")
	}
	betaEndpoint, err := connect(identifier, version, beta, ignores, false, prompter)
	if err != nil {
		alphaEndpoint.close()
		return nil, errors.Wrap(err, "unable to connect to beta")
	}

	// Create the session and archive.
	session := &Session{
		Identifier:           identifier,
		Version:              version,
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
		alphaEndpoint.close()
		betaEndpoint.close()
		return nil, errors.Wrap(err, "unable to compute session path")
	}
	archivePath, err := pathForArchive(session.Identifier)
	if err != nil {
		alphaEndpoint.close()
		betaEndpoint.close()
		return nil, errors.Wrap(err, "unable to compute archive path")
	}

	// Save components to disk.
	if err := encoding.MarshalAndSaveProtobuf(sessionPath, session); err != nil {
		alphaEndpoint.close()
		betaEndpoint.close()
		return nil, errors.Wrap(err, "unable to save session")
	}
	if err := encoding.MarshalAndSaveProtobuf(archivePath, archive); err != nil {
		os.Remove(sessionPath)
		alphaEndpoint.close()
		betaEndpoint.close()
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
	go controller.run(context, alphaEndpoint, betaEndpoint)

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
	alpha, alphaConnectErr := connect(
		c.session.Identifier,
		c.session.Version,
		c.session.Alpha,
		c.session.Ignores,
		true,
		prompter,
	)
	beta, betaConnectErr := connect(
		c.session.Identifier,
		c.session.Version,
		c.session.Beta,
		c.session.Ignores,
		false,
		prompter,
	)

	// Start the synchronization loop with what we have.
	context, cancel := contextpkg.WithCancel(contextpkg.Background())
	c.cancel = cancel
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

func (c *controller) run(context contextpkg.Context, alpha, beta endpoint) {
	// Defer resource and state cleanup.
	defer func() {
		// Close any endpoints. These might be non-nil if the runloop was
		// cancelled while partially connected rather than after sync failure.
		if alpha != nil {
			alpha.close()
		}
		if beta != nil {
			beta.close()
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
				alpha, _ = reconnect(
					context,
					c.session.Identifier,
					c.session.Version,
					c.session.Alpha,
					c.session.Ignores,
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
				beta, _ = reconnect(
					context,
					c.session.Identifier,
					c.session.Version,
					c.session.Beta,
					c.session.Ignores,
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

		// Close the endpoints.
		alpha.close()
		alpha = nil
		beta.close()
		beta = nil

		// Reset the synchronization state, but propagate the error that caused
		// failure.
		c.stateLock.Lock()
		c.state = SynchronizationState{
			LastError: err.Error(),
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

func (c *controller) synchronize(context contextpkg.Context, alpha, beta endpoint) error {
	// Load the archive and extract the ancestor.
	archive := &Archive{}
	if err := encoding.LoadAndUnmarshalProtobuf(c.archivePath, archive); err != nil {
		return errors.Wrap(err, "unable to load archive")
	}
	ancestor := archive.Root

	// Loop until there is a synchronization error. We always skip polling on
	// the first time through the loop because changes may have occurred while
	// we were halted. We also skip polling in the event that an endpoint asks
	// for a scan retry.
	skipPolling := true
	for {
		// Unless we've been requested to skip polling, wait for a dirty state
		// while monitoring for cancellation. If we've been requested to skip
		// polling, it should only be for one iteration.
		// TODO: Should we try to drain the alpha and beta pollers before
		// continuing to avoid excessive synchronization?
		if !skipPolling {
			// Update status to watching.
			c.stateLock.Lock()
			c.state.Status = SynchronizationStatusWatching
			c.stateLock.Unlock()

			select {
			case _, ok := <-alpha.poller():
				if !ok {
					return errors.New("alpha watch broken")
				}
			case _, ok := <-beta.poller():
				if !ok {
					return errors.New("beta watch broken")
				}
			case <-context.Done():
				return errors.New("cancelled during polling")
			}
		} else {
			skipPolling = false
		}

		// Update status to scanning.
		c.stateLock.Lock()
		c.state.Status = SynchronizationStatusScanning
		c.stateLock.Unlock()

		// Perform scans.
		alphaSnapshot, alphaTryAgain, alphaScanErr := alpha.scan(ancestor)
		if alphaScanErr != nil {
			return errors.Wrap(alphaScanErr, "alpha scan error")
		}
		betaSnapshot, betaTryAgain, betaScanErr := beta.scan(ancestor)
		if betaScanErr != nil {
			return errors.Wrap(betaScanErr, "beta scan error")
		}

		// Watch for retry requests.
		// TODO: Should we eventually abort synchronization after a certain
		// number of consecutive scan retries?
		if alphaTryAgain || betaTryAgain {
			// Update status to waiting for rescan.
			c.stateLock.Lock()
			c.state.Status = SynchronizationStatusWaitingForRescan
			c.stateLock.Unlock()

			// Wait before trying to rescan, but watch for cancellation.
			select {
			case <-time.After(rescanWaitDuration):
			case <-context.Done():
				return errors.New("cancelled during rescan wait")
			}

			// Retry.
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

		// Stage files on alpha.
		stagingPaths, stagingEntries, stagingError := stagingPathsForChanges(alphaTransitions)
		if stagingError != nil {
			return errors.Wrap(stagingError, "unable to determine paths for staging on alpha")
		}
		stagingPaths, stagingSignatures, stagingReceiver, stagingError := alpha.stage(stagingPaths, stagingEntries)
		if stagingError != nil {
			return errors.Wrap(stagingError, "unable to begin staging on alpha")
		}
		stagingReceiver, stagingError = rsync.NewMonitoringReceiver(stagingReceiver, stagingPaths, func(status rsync.ReceivingStatus) error {
			c.stateLock.Lock()
			c.state.AlphaStaging = status
			c.stateLock.Unlock()
			return nil
		})
		if stagingError != nil {
			return errors.Wrap(stagingError, "unable to create monitoring receiver for staging on alpha")
		}
		stagingReceiver = rsync.NewPreemptableReceiver(stagingReceiver, context)
		if stagingError = beta.supply(stagingPaths, stagingSignatures, stagingReceiver); stagingError != nil {
			return errors.Wrap(stagingError, "unable to stage files on alpha")
		}

		// Stage files on beta.
		stagingPaths, stagingEntries, stagingError = stagingPathsForChanges(betaTransitions)
		if stagingError != nil {
			return errors.Wrap(stagingError, "unable to determine paths for staging on beta")
		}
		stagingPaths, stagingSignatures, stagingReceiver, stagingError = beta.stage(stagingPaths, stagingEntries)
		if stagingError != nil {
			return errors.Wrap(stagingError, "unable to begin staging on beta")
		}
		stagingReceiver, stagingError = rsync.NewMonitoringReceiver(stagingReceiver, stagingPaths, func(status rsync.ReceivingStatus) error {
			c.stateLock.Lock()
			c.state.BetaStaging = status
			c.stateLock.Unlock()
			return nil
		})
		if stagingError != nil {
			return errors.Wrap(stagingError, "unable to create monitoring receiver for staging on beta")
		}
		stagingReceiver = rsync.NewPreemptableReceiver(stagingReceiver, context)
		if stagingError = alpha.supply(stagingPaths, stagingSignatures, stagingReceiver); stagingError != nil {
			return errors.Wrap(stagingError, "unable to stage files on beta")
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
		alphaChanges, alphaProblems, alphaTransitionErr := alpha.transition(alphaTransitions)
		betaChanges, betaProblems, betaTransitionErr := beta.transition(betaTransitions)

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
		if newAncestor, err := sync.Apply(ancestor, ancestorChanges); err != nil {
			return errors.Wrap(err, "unable to propagate changes to ancestor")
		} else {
			ancestor = newAncestor
		}

		// Validate the new ancestor before saving it to ensure that our
		// reconciliation logic doesn't have any flaws.
		if err := ancestor.EnsureValid(); err != nil {
			return errors.Wrap(err, "new ancestor is invalid")
		}

		// Save the ancestor.
		archive.Root = ancestor
		if err := encoding.MarshalAndSaveProtobuf(c.archivePath, archive); err != nil {
			return errors.Wrap(err, "unable to save ancestor")
		}

		// Now check for transition errors.
		if alphaTransitionErr != nil {
			return errors.Wrap(alphaTransitionErr, "unable to apply changes to alpha")
		} else if betaTransitionErr != nil {
			return errors.Wrap(alphaTransitionErr, "unable to apply changes to beta")
		}

		// After a successful synchronization cycle, clear any synchronization
		// error.
		c.stateLock.Lock()
		c.state.LastError = ""
		c.stateLock.Unlock()
	}
}
