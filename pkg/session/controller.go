package session

import (
	contextpkg "context"
	"os"
	syncpkg "sync"
	"time"

	"github.com/pkg/errors"

	"github.com/google/uuid"

	"github.com/golang/protobuf/ptypes"

	"github.com/havoc-io/mutagen/pkg/encoding"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/state"
	"github.com/havoc-io/mutagen/pkg/sync"
	"github.com/havoc-io/mutagen/pkg/url"
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
	state *State
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

func newSession(tracker *state.Tracker, alpha, beta *url.URL, configuration *Configuration, prompter string) (*controller, error) {
	// Create a snapshot of the global configuration.
	globalConfiguration, err := snapshotGlobalConfiguration()
	if err != nil {
		return nil, errors.Wrap(err, "unable to snapshot global configuration")
	}

	// Create an effective merged configuration.
	configuration = MergeConfigurations(configuration, globalConfiguration)

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

	// Attempt to connect. Session creation is only allowed after if successful.
	alphaEndpoint, err := connect(identifier, version, alpha, configuration, true, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to alpha")
	}
	betaEndpoint, err := connect(identifier, version, beta, configuration, false, prompter)
	if err != nil {
		alphaEndpoint.shutdown()
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
	}
	archive := &sync.Archive{}

	// Compute session and archive paths.
	sessionPath, err := pathForSession(session.Identifier)
	if err != nil {
		alphaEndpoint.shutdown()
		betaEndpoint.shutdown()
		return nil, errors.Wrap(err, "unable to compute session path")
	}
	archivePath, err := pathForArchive(session.Identifier)
	if err != nil {
		alphaEndpoint.shutdown()
		betaEndpoint.shutdown()
		return nil, errors.Wrap(err, "unable to compute archive path")
	}

	// Save components to disk.
	if err := encoding.MarshalAndSaveProtobuf(sessionPath, session); err != nil {
		alphaEndpoint.shutdown()
		betaEndpoint.shutdown()
		return nil, errors.Wrap(err, "unable to save session")
	}
	if err := encoding.MarshalAndSaveProtobuf(archivePath, archive); err != nil {
		os.Remove(sessionPath)
		alphaEndpoint.shutdown()
		betaEndpoint.shutdown()
		return nil, errors.Wrap(err, "unable to save archive")
	}

	// Create the controller.
	controller := &controller{
		sessionPath: sessionPath,
		archivePath: archivePath,
		stateLock:   state.NewTrackingLock(tracker),
		session:     session,
		state: &State{
			Session: session,
		},
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

	// Load and validate the session.
	session := &Session{}
	if err := encoding.LoadAndUnmarshalProtobuf(sessionPath, session); err != nil {
		return nil, errors.Wrap(err, "unable to load session configuration")
	} else if err = session.EnsureValid(); err != nil {
		return nil, errors.Wrap(err, "invalid session found on disk")
	}

	// Create the controller.
	controller := &controller{
		sessionPath: sessionPath,
		archivePath: archivePath,
		stateLock:   state.NewTrackingLock(tracker),
		session:     session,
		state: &State{
			Session: session,
		},
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

func (c *controller) currentState() *State {
	// Lock the session state and defer its release. It's very important that we
	// unlock without a notification here, otherwise we'd trigger an infinite
	// cycle of list/notify.
	c.stateLock.Lock()
	defer c.stateLock.UnlockWithoutNotify()

	// Perform a (pseudo) deep copy of the state.
	return c.state.Copy()
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
		c.session.Identifier,
		c.session.Version,
		c.session.Alpha,
		c.session.Configuration,
		true,
		prompter,
	)
	c.stateLock.Lock()
	c.state.AlphaConnected = (alpha != nil)
	c.stateLock.Unlock()

	// Attempt to connect to beta.
	c.stateLock.Lock()
	c.state.Status = Status_ConnectingBeta
	c.stateLock.Unlock()
	beta, betaConnectErr := connect(
		c.session.Identifier,
		c.session.Version,
		c.session.Beta,
		c.session.Configuration,
		false,
		prompter,
	)
	c.stateLock.Lock()
	c.state.BetaConnected = (beta != nil)
	c.stateLock.Unlock()

	// Start the synchronization loop with what we have. Alpha or beta may have
	// failed to connect (and be nil), but in any case that'll just make the run
	// loop keep trying to connect.
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
		// Shutdown any endpoints. These might be non-nil if the runloop was
		// cancelled while partially connected rather than after sync failure.
		if alpha != nil {
			alpha.shutdown()
		}
		if beta != nil {
			beta.shutdown()
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
					c.session.Identifier,
					c.session.Version,
					c.session.Alpha,
					c.session.Configuration,
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
					c.session.Identifier,
					c.session.Version,
					c.session.Beta,
					c.session.Configuration,
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
		alpha.shutdown()
		alpha = nil
		beta.shutdown()
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

func (c *controller) synchronize(context contextpkg.Context, alpha, beta endpoint) error {
	// Load the archive and extract the ancestor.
	archive := &sync.Archive{}
	if err := encoding.LoadAndUnmarshalProtobuf(c.archivePath, archive); err != nil {
		return errors.Wrap(err, "unable to load archive")
	} else if err = archive.Root.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid archive found on disk")
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
		if !skipPolling {
			// Update status to watching.
			c.stateLock.Lock()
			c.state.Status = Status_Watching
			c.stateLock.Unlock()

			// Create a polling context that we can cancel. We don't make it a
			// subcontext of our own cancellation context because it's easier to
			// just track cancellation there separately.
			pollContext, pollCancel := contextpkg.WithCancel(contextpkg.Background())

			// Start alpha polling.
			αPollResults := make(chan error, 1)
			go func() {
				αPollResults <- alpha.poll(pollContext)
			}()

			// Start beta polling.
			βPollResults := make(chan error, 1)
			go func() {
				βPollResults <- beta.poll(pollContext)
			}()

			// Wait for either poll to return an event or an error, or for
			// cancellation. In any of these cases, cancel polling and ensure
			// that both polling operations have completed.
			var αPollErr, βPollErr error
			cancelled := false
			select {
			case αPollErr = <-αPollResults:
				pollCancel()
				βPollErr = <-βPollResults
			case βPollErr = <-βPollResults:
				pollCancel()
				αPollErr = <-αPollResults
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

		// Scan both endpoints in parallel and check for errors.
		c.stateLock.Lock()
		c.state.Status = Status_Scanning
		c.stateLock.Unlock()
		var αSnapshot, βSnapshot *sync.Entry
		var αPreservesExecutability, βPreservesExecutability bool
		var αScanErr, βScanErr error
		var αTryAgain, βTryAgain bool
		scanDone := &syncpkg.WaitGroup{}
		scanDone.Add(2)
		go func() {
			αSnapshot, αPreservesExecutability, αScanErr, αTryAgain = alpha.scan(ancestor)
			scanDone.Done()
		}()
		go func() {
			βSnapshot, βPreservesExecutability, βScanErr, βTryAgain = beta.scan(ancestor)
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

		// Watch for retry requests.
		// TODO: Should we eventually abort synchronization after a certain
		// number of consecutive scan retries?
		if αTryAgain || βTryAgain {
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

			// Retry.
			skipPolling = true
			continue
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

		// Perform reconciliation and record conflicts.
		ancestorChanges, αTransitions, βTransitions, conflicts := sync.Reconcile(
			ancestor, αSnapshot, βSnapshot,
		)
		c.stateLock.Lock()
		c.state.Conflicts = conflicts
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
		if paths, entries, err := sync.TransitionDependencies(αTransitions); err != nil {
			return errors.Wrap(err, "unable to determine paths for staging on alpha")
		} else if len(paths) > 0 {
			paths, signatures, receiver, err := alpha.stage(paths, entries)
			if err != nil {
				return errors.Wrap(err, "unable to begin staging on alpha")
			}
			if len(paths) > 0 {
				receiver = rsync.NewMonitoringReceiver(receiver, paths, monitor)
				receiver = rsync.NewPreemptableReceiver(receiver, context)
				if err = beta.supply(paths, signatures, receiver); err != nil {
					return errors.Wrap(err, "unable to stage files on alpha")
				}
			}
		}

		// Stage files on beta.
		c.stateLock.Lock()
		c.state.Status = Status_StagingBeta
		c.stateLock.Unlock()
		if paths, entries, err := sync.TransitionDependencies(βTransitions); err != nil {
			return errors.Wrap(err, "unable to determine paths for staging on beta")
		} else if len(paths) > 0 {
			paths, signatures, receiver, err := beta.stage(paths, entries)
			if err != nil {
				return errors.Wrap(err, "unable to begin staging on beta")
			}
			if len(paths) > 0 {
				receiver = rsync.NewMonitoringReceiver(receiver, paths, monitor)
				receiver = rsync.NewPreemptableReceiver(receiver, context)
				if err = alpha.supply(paths, signatures, receiver); err != nil {
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
		var αTransitionErr, βTransitionErr error
		var αChanges, βChanges []*sync.Change
		transitionDone := &syncpkg.WaitGroup{}
		transitionDone.Add(2)
		go func() {
			if len(αTransitions) > 0 {
				αResults, αProblems, αTransitionErr = alpha.transition(αTransitions)
				if αTransitionErr == nil {
					for t, transition := range αTransitions {
						αChanges = append(αChanges, &sync.Change{Path: transition.Path, New: αResults[t]})
					}
				}
			}
			transitionDone.Done()
		}()
		go func() {
			if len(βTransitions) > 0 {
				βResults, βProblems, βTransitionErr = beta.transition(βTransitions)
				if βTransitionErr == nil {
					for t, transition := range βTransitions {
						βChanges = append(βChanges, &sync.Change{Path: transition.Path, New: βResults[t]})
					}
				}
			}
			transitionDone.Done()
		}()
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

		// After a successful synchronization cycle, increment the cycle count
		// and clear any synchronization error.
		c.stateLock.Lock()
		c.state.SuccessfulSynchronizationCycles++
		c.state.LastError = ""
		c.stateLock.Unlock()
	}
}
