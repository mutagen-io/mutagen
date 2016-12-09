package session

import (
	"os"
	"time"

	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	uuid "github.com/satori/go.uuid"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/encoding"
	"github.com/havoc-io/mutagen/sync"
	"github.com/havoc-io/mutagen/url"
)

const (
	autoReconnectInterval = 30 * time.Second
)

type controller struct {
	stateTracker              *stateTracker
	session                   *Session
	archive                   *Archive
	sessionPath               string
	archivePath               string
	lifecycleParametersLocked bool
	context                   context.Context
	cancel                    context.CancelFunc
	done                      chan struct{}
	alphaConnection           *grpc.ClientConn
	betaConnection            *grpc.ClientConn
	status                    SynchronizationStatus
	message                   string
	conflicts                 []*sync.Conflict
	problems                  []*sync.Problem
}

func newSession(
	stateTracker *stateTracker,
	dialContext context.Context,
	alpha, beta *url.URL,
	prompter string,
) (*controller, error) {
	// Attempt to dial. Session creation is only allowed after a successful
	// dial.
	alphaConnection, err := agent.Dial(dialContext, alpha, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to alpha")
	}
	betaConnection, err := agent.Dial(dialContext, beta, prompter)
	if err != nil {
		alphaConnection.Close()
		return nil, errors.Wrap(err, "unable to connect to beta")
	}

	// Create the session and archive.
	creationTime := time.Now()
	session := &Session{
		Identifier:           uuid.NewV4().String(),
		Version:              SessionVersion_Version1,
		CreationTime:         &creationTime,
		CreatingVersionMajor: mutagen.VersionMajor,
		CreatingVersionMinor: mutagen.VersionMinor,
		CreatingVersionPatch: mutagen.VersionPatch,
		Alpha:                alpha,
		Beta:                 beta,
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

	// Create controller and start a synchronization loop.
	context, cancel := context.WithCancel(context.Background())
	controller := &controller{
		stateTracker:    stateTracker,
		session:         session,
		archive:         archive,
		sessionPath:     sessionPath,
		archivePath:     archivePath,
		context:         context,
		cancel:          cancel,
		done:            make(chan struct{}),
		alphaConnection: alphaConnection,
		betaConnection:  betaConnection,
	}
	go controller.synchronize()

	// Success.
	return controller, nil
}

func loadSession(stateTracker *stateTracker, identifier string) (*controller, error) {
	// Compute session and archive paths.
	sessionPath, err := pathForSession(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute session path")
	}
	archivePath, err := pathForArchive(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute archive path")
	}

	// Load the session and archive.
	session := &Session{}
	if err := encoding.LoadAndUnmarshalProtobuf(sessionPath, session); err != nil {
		return nil, errors.Wrap(err, "unable to load session configuration")
	}
	archive := &Archive{}
	if err := encoding.LoadAndUnmarshalProtobuf(archivePath, archive); err != nil {
		return nil, errors.Wrap(err, "unable to load archive")
	}

	// Create the controller.
	controller := &controller{
		stateTracker: stateTracker,
		session:      session,
		archive:      archive,
		sessionPath:  sessionPath,
		archivePath:  archivePath,
	}

	// If the session isn't marked as paused, start a synchronization loop.
	if !session.Paused {
		context, cancel := context.WithCancel(context.Background())
		controller.context = context
		controller.cancel = cancel
		controller.done = make(chan struct{})
		go controller.synchronize()
	}

	// Success.
	return controller, nil
}

func (c *controller) synchronize() {
	// Register a state cleanup routine when the loop exits.
	defer func() {
		// Lock the state.
		c.stateTracker.lock()

		// Close out any connections.
		if c.alphaConnection != nil {
			c.alphaConnection.Close()
			c.alphaConnection = nil
		}
		if c.betaConnection != nil {
			c.betaConnection.Close()
			c.betaConnection = nil
		}

		// Reset state.
		c.status = SynchronizationStatus_Idle
		c.message = ""
		c.conflicts = nil
		c.problems = nil

		// Unlock the state and send a change notification.
		c.stateTracker.notifyOfChangesAndUnlock()

		// Signal termination.
		close(c.done)
	}()

	// Loop until cancelled.
	// TODO: Implement.
	<-c.context.Done()
}

// TODO: Note that this is the one function that must be called with the state
// lock held.
func (c *controller) state() *SessionState {
	// Create a blank result.
	result := &SessionState{
		Session: &Session{},
	}

	// Make a shallow copy of the session. It can be mutated once the state lock
	// is released, but its field values are immutable, so the copy will be
	// unchanged.
	*result.Session = *c.session

	// Copy state parameters. Problem and conflict slices are technically
	// mutable, but they (and their contents) are treated as immutable, they are
	// simply replaced completely.
	result.AlphaConnected = (c.alphaConnection != nil)
	result.BetaConnected = (c.betaConnection != nil)
	result.Status = c.status
	result.Message = c.message
	result.Conflicts = c.conflicts
	result.Problems = c.problems

	// Done.
	return result
}

func (c *controller) pause() error {
	// Grab the state lock.
	c.stateTracker.lock()

	// Check if the lifecycle parameters are locked. If so, abort, otherwise
	// lock them.
	if c.lifecycleParametersLocked {
		c.stateTracker.unlock()
		return errors.New("session operation in-progress")
	}
	c.lifecycleParametersLocked = true

	// If there's a synchronization loop running, release the state lock, cancel
	// it, wait for it to complete, and reacquire the state lock. It's safe to
	// release the state lock while doing this since we've locked the lifecycle
	// parameters.
	if c.done != nil {
		c.stateTracker.unlock()
		c.cancel()
		<-c.done
		c.stateTracker.lock()
	}

	// Clear out synchronization loop lifecycle parameters.
	c.context = nil
	c.cancel = nil
	c.done = nil

	// Mark the session as paused and save its configuration.
	c.session.Paused = true
	saveErr := encoding.MarshalAndSaveProtobuf(c.sessionPath, c.session)

	// Release the lock on the lifecycle parameters.
	c.lifecycleParametersLocked = false

	// Release the state lock and notify of changes. The notification is
	// necessary in this case because we changed the paused status on the
	// session.
	c.stateTracker.notifyOfChangesAndUnlock()

	// Check if there were any problems saving the session state.
	if saveErr != nil {
		return errors.Wrap(saveErr, "unable to save session configuration")
	}

	// Success.
	return nil
}

func (c *controller) resume(dialContext context.Context, prompter string) error {
	// TODO: Implement.
	return errors.New("not implemented")
}

func (c *controller) stop(wipe bool) error {
	// Grab the state lock.
	c.stateTracker.lock()

	// Check if the lifecycle parameters are locked. If so, abort, otherwise
	// lock them.
	if c.lifecycleParametersLocked {
		c.stateTracker.unlock()
		return errors.New("session operation in-progress")
	}
	c.lifecycleParametersLocked = true

	// If there's a synchronization loop running, release the state lock, cancel
	// it, wait for it to complete, and reacquire the state lock. It's safe to
	// release the state lock while doing this since we've locked the lifecycle
	// parameters.
	if c.done != nil {
		c.stateTracker.unlock()
		c.cancel()
		<-c.done
		c.stateTracker.lock()
	}

	// Clear out synchronization loop lifecycle parameters. We never release our
	// lock on them, because the session has been stopped and no subsequent
	// resume attempts should succeed.
	c.context = nil
	c.cancel = nil
	c.done = nil

	// Wipe out the session configuration and archive if requested. If these
	// operations fail, there's not really much point in reporting them back to
	// the caller - the session should still be deregistered at that point. If
	// both fail, the session will just be reloaded next time the service
	// starts. If one fails, the session will never be reloaded. Either is an
	// okay scenario, which is why there's no point in reporting this to the
	// caller. Even if we wanted to, it just complicates things, because then we
	// have to use a sentinel error to indicate a failure here vs. failure due
	// to the lifecycle parameters being locked, and then it's not clear if the
	// caller should attempt to restart or leave the session registered or what.
	// Always deregistering it is the simplest and most robust solution.
	// TODO: What we should do here is add some logging in the event that these
	// operations fail.
	if wipe {
		os.Remove(c.sessionPath)
		os.Remove(c.archivePath)
	}

	// Release the state lock. We don't need to notify of any changes here.
	c.stateTracker.unlock()

	// Success.
	return nil
}
