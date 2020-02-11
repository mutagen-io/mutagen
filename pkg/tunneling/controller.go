package tunneling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/pion/webrtc/v2"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/mutagenio"
	"github.com/mutagen-io/mutagen/pkg/prompting"
	"github.com/mutagen-io/mutagen/pkg/random"
	"github.com/mutagen-io/mutagen/pkg/state"
	"github.com/mutagen-io/mutagen/pkg/tunneling/webrtcutil"
)

const (
	// tunnelCreateTimeout is the maximum amount of time allowed for tunnel
	// creation.
	tunnelCreateTimeout = 5 * time.Second
	// tunnelConnectRetryDelayTime is the amount of time to wait before retrying
	// a controller.connect call after experiencing a failure with
	// ErrorSeverityDelayedRecoverable.
	tunnelConnectRetryDelayTime = 5 * time.Second
)

// dialRequest represents a request for a new data channel.
type dialRequest struct {
	// ctx is the context regulating the request.
	ctx context.Context
	// results is the channel through which any resulting connection should be
	// provided.
	results chan net.Conn
	// errors is the channel through which any resulting error should be
	// provided.
	errors chan error
}

// controller manages and executes a single tunnel.
type controller struct {
	// logger is the controller logger.
	logger *logging.Logger
	// tunnelPath is the path to the serialized tunnel.
	tunnelPath string
	// stateLock guards and tracks changes to the tunnel member's Paused field
	// and the state member.
	stateLock *state.TrackingLock
	// tunnel encodes the associated tunnel client metadata. It is considered
	// static and safe for concurrent access except for its Paused field, for
	// which the stateLock member should be held. It should be saved to disk any
	// time it is modified.
	tunnel *Tunnel
	// state represents the current tunnel state.
	state *State
	// dialRequests is used pass dial requests to the serving loop.
	dialRequests chan dialRequest
	// lifecycleLock guards setting of the disabled, cancel, and done members.
	// Access to these members is allowed for the connection loop without
	// holding the lock. Any code wishing to set these members should first
	// acquire the lock, then cancel the connection loop, and wait for it to
	// complete before making any such changes.
	lifecycleLock sync.Mutex
	// disabled indicates that no more changes to the connection loop lifecycle
	// are allowed (i.e. no more connection loops can be started for this
	// controller). This is used by terminate and shutdown. It should only be
	// set to true once any existing connection loop has been stopped.
	disabled bool
	// cancel cancels the connection loop execution context. It should be nil if
	// and only if there is no connection loop running.
	cancel context.CancelFunc
	// done will be closed by the current connection loop when it exits.
	done chan struct{}
}

// newTunnel creates a new tunnel and corresponding controller. It also returns
// the parameters needed to host the tunnel.
func newTunnel(
	ctx context.Context,
	logger *logging.Logger,
	tracker *state.Tracker,
	configuration *Configuration,
	name string,
	labels map[string]string,
	paused bool,
	prompter string,
) (*controller, *TunnelHostCredentials, error) {
	// Update status.
	prompting.Message(prompter, "Creating tunnel...")

	// Set the tunnel version.
	version := Version_Version1

	// Compute the creation time and convert it to Protocol Buffers format.
	creationTime := time.Now()
	creationTimeProto, err := ptypes.TimestampProto(creationTime)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to convert creation time format: %w", err)
	}

	// Generate the tunnel secret.
	secret, err := random.New(version.secretLength())
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate tunnel secret: %w", err)
	}

	// Attempt to create the tunnel via the API.
	ctx, cancel := context.WithTimeout(ctx, tunnelCreateTimeout)
	defer cancel()
	identifier, hostToken, clientToken, err := mutagenio.TunnelCreate(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Create the host endpoint parameters.
	hostCredentials := &TunnelHostCredentials{
		Identifier:           identifier,
		Version:              version,
		CreationTime:         creationTimeProto,
		CreatingVersionMajor: mutagen.VersionMajor,
		CreatingVersionMinor: mutagen.VersionMinor,
		CreatingVersionPatch: mutagen.VersionPatch,
		Token:                hostToken,
		Secret:               secret,
		Configuration:        configuration,
	}

	// Create the tunnel configuration.
	tunnel := &Tunnel{
		Identifier:           identifier,
		Version:              version,
		CreationTime:         creationTimeProto,
		CreatingVersionMajor: mutagen.VersionMajor,
		CreatingVersionMinor: mutagen.VersionMinor,
		CreatingVersionPatch: mutagen.VersionPatch,
		Token:                clientToken,
		Secret:               secret,
		Configuration:        configuration,
		Name:                 name,
		Labels:               labels,
		Paused:               paused,
	}

	// Compute the tunnel path.
	tunnelPath, err := pathForTunnel(tunnel.Identifier)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to compute tunnel path: %w", err)
	}

	// Save the tunnel to disk.
	if err := encoding.MarshalAndSaveProtobuf(tunnelPath, tunnel); err != nil {
		return nil, nil, fmt.Errorf("unable to save tunnel: %w", err)
	}

	// Create the controller. We create our own sublogger since we're the one
	// that knows the identifier.
	controller := &controller{
		logger:     logger.Sublogger(identifier),
		tunnelPath: tunnelPath,
		stateLock:  state.NewTrackingLock(tracker),
		tunnel:     tunnel,
		state: &State{
			Tunnel: tunnel,
		},
		dialRequests: make(chan dialRequest),
	}

	// If the tunnel isn't being created paused, then start a connection loop.
	if !paused {
		logger.Info("Starting tunnel connection loop")
		ctx, cancel := context.WithCancel(context.Background())
		controller.cancel = cancel
		controller.done = make(chan struct{})
		go controller.run(ctx)
	}

	// Success.
	logger.Info("Tunnel initialized")
	return controller, hostCredentials, nil
}

// loadTunnel loads an existing tunnel and creates a corresponding controller.
func loadTunnel(logger *logging.Logger, tracker *state.Tracker, identifier string) (*controller, error) {
	// Compute the tunnel path.
	tunnelPath, err := pathForTunnel(identifier)
	if err != nil {
		return nil, fmt.Errorf("unable to compute tunnel path: %w", err)
	}

	// Load and validate the tunnel.
	tunnel := &Tunnel{}
	if err := encoding.LoadAndUnmarshalProtobuf(tunnelPath, tunnel); err != nil {
		return nil, fmt.Errorf("unable to load tunnel configuration: %w", err)
	}
	if err := tunnel.EnsureValid(); err != nil {
		return nil, fmt.Errorf("invalid tunnel found on disk: %w", err)
	}

	// Create the controller.
	controller := &controller{
		logger:     logger,
		tunnelPath: tunnelPath,
		stateLock:  state.NewTrackingLock(tracker),
		tunnel:     tunnel,
		state: &State{
			Tunnel: tunnel,
		},
		dialRequests: make(chan dialRequest),
	}

	// If the tunnel isn't marked as paused, start a connection loop.
	if !tunnel.Paused {
		ctx, cancel := context.WithCancel(context.Background())
		controller.cancel = cancel
		controller.done = make(chan struct{})
		go controller.run(ctx)
	}

	// Success.
	logger.Info("Tunnel loaded")
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

// resume attempts to resume the tunnel if it isn't currently connected.
func (c *controller) resume(_ context.Context, prompter string) error {
	// Update status.
	prompting.Message(prompter, fmt.Sprintf("Resuming tunnel %s...", c.tunnel.Identifier))

	// Lock the controller's lifecycle and defer its release.
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Don't allow any resume operations if the controller is disabled.
	if c.disabled {
		return errors.New("controller disabled")
	}

	// Check if there's an existing connection loop, which indicates that the
	// tunnel is already unpaused. If so, then there's nothing else to do.
	if c.cancel != nil {
		return nil
	}

	// Mark the tunnel as unpaused and save it to disk.
	c.stateLock.Lock()
	c.tunnel.Paused = false
	saveErr := encoding.MarshalAndSaveProtobuf(c.tunnelPath, c.tunnel)
	c.stateLock.Unlock()

	// Start the connection loop.
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.done = make(chan struct{})
	go c.run(ctx)

	// Report any errors. Since we always want to start a connection loop, even
	// in the case of some partial failure, we wait until the end to report
	// errors.
	if saveErr != nil {
		return fmt.Errorf("unable to save session: %w", saveErr)
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

// halt halts the tunnel with the specified behavior.
func (c *controller) halt(_ context.Context, mode controllerHaltMode, prompter string) error {
	// Update status.
	prompting.Message(prompter, fmt.Sprintf("%s tunnel %s...", mode.description(), c.tunnel.Identifier))

	// Lock the controller's lifecycle and defer its release.
	c.lifecycleLock.Lock()
	defer c.lifecycleLock.Unlock()

	// Don't allow any additional halt operations if the controller is disabled,
	// because either this tunnel is being terminated or the service is shutting
	// down, and in either case there is no point in halting.
	if c.disabled {
		return errors.New("controller disabled")
	}

	// Kill any existing connection loop.
	if c.cancel != nil {
		// Cancel the connection loop and wait for it to finish.
		c.cancel()
		<-c.done

		// Nil out any lifecycle state.
		c.cancel = nil
		c.done = nil
	}

	// Handle based on the halt mode.
	if mode == controllerHaltModePause {
		// Mark the tunnel as paused and save it.
		c.stateLock.Lock()
		c.tunnel.Paused = true
		saveErr := encoding.MarshalAndSaveProtobuf(c.tunnelPath, c.tunnel)
		c.stateLock.Unlock()
		if saveErr != nil {
			return fmt.Errorf("unable to save tunnel: %w", saveErr)
		}
	} else if mode == controllerHaltModeShutdown {
		// Disable the controller.
		c.disabled = true
	} else if mode == controllerHaltModeTerminate {
		// Disable the controller.
		c.disabled = true

		// Wipe the tunnel information from disk.
		tunnelRemoveErr := os.Remove(c.tunnelPath)
		if tunnelRemoveErr != nil {
			return fmt.Errorf("unable to remove tunnel from disk: %w", tunnelRemoveErr)
		}
	} else {
		panic("invalid halt mode specified")
	}

	// Success.
	return nil
}

// run is the main runloop for the controller.
func (c *controller) run(ctx context.Context) {
	// Track any active peer connection.
	var peerConnection *webrtc.PeerConnection

	// Defer resource and state cleanup.
	defer func() {
		// Log the termination.
		c.logger.Info("Run loop terminating")

		// Shut down any active peer connection.
		if peerConnection != nil {
			peerConnection.Close()
		}

		// Reset the state.
		c.stateLock.Lock()
		c.state = &State{
			Tunnel: c.tunnel,
		}
		c.stateLock.Unlock()

		// Signal completion.
		close(c.done)
	}()

	// Loop until cancelled or until an unrecoverable error has occurred.
	var unrecoverableErr error
	for {
		// Update the state to connecting.
		c.stateLock.Lock()
		c.state.Status = Status_Connecting
		c.stateLock.Unlock()

		// Create the peer connection.
		c.logger.Info("Attempting a peer connection")
		peerConnection, peerConnectionFailures, severity, err := c.connect(ctx)
		if err != nil {
			// If this is an unrecoverable error, then halt.
			if severity == ErrorSeverityUnrecoverable {
				c.logger.Info("Peer connection unrecoverable error:", err)
				unrecoverableErr = err
				break
			} else {
				c.logger.Info("Peer connection error:", err)
			}

			// Update the error state.
			c.stateLock.Lock()
			c.state = &State{
				Tunnel:    c.tunnel,
				LastError: err.Error(),
			}
			c.stateLock.Unlock()

			// If the context has been cancelled, then terminate.
			select {
			case <-ctx.Done():
				return
			default:
			}

			// If this is a delayed recovery error, then wait before retrying.
			if severity == ErrorSeverityDelayedRecoverable {
				c.logger.Info("Waiting to attempt reconnection")
				select {
				case <-time.After(tunnelConnectRetryDelayTime):
				case <-ctx.Done():
					return
				}
			}

			// Retry the connection.
			continue
		}

		// Upate the state to connected and clear any previous error.
		c.logger.Info("Peer connection successful")
		c.stateLock.Lock()
		c.state.Status = Status_Connected
		c.state.LastError = ""
		c.stateLock.Unlock()

		// Perform serving.
		c.logger.Info("Serving peer connection")
		err = c.serve(ctx, peerConnection, peerConnectionFailures)

		// Close the peer connection.
		c.logger.Info("Closing peer connection due to error:", err)
		peerConnection.Close()
		peerConnection = nil

		// Reset the tunneling state, but propagate the error that caused
		// failure.
		c.stateLock.Lock()
		c.state = &State{
			Tunnel:    c.tunnel,
			LastError: err.Error(),
		}
		c.stateLock.Unlock()

		// If the context has been cancelled, then terminate.
		select {
		case <-ctx.Done():
			return
		default:
		}
	}

	// At this point an unrecoverable error has occurred, so just halt and wait
	// for cancellation.
	c.stateLock.Lock()
	c.state = &State{
		Tunnel:    c.tunnel,
		Status:    Status_HaltedOnUnrecoverableError,
		LastError: unrecoverableErr.Error(),
	}
	c.stateLock.Unlock()
	<-ctx.Done()
}

func (c *controller) connect(ctx context.Context) (*webrtc.PeerConnection, chan error, ErrorSeverity, error) {
	// Load the WebRTC API.
	api, err := loadWebRTCAPI()
	if err != nil {
		return nil, nil, ErrorSeverityUnrecoverable, fmt.Errorf("unable to initialize tunnel network API: %w", err)
	}

	// Create an unconnected peer connection.
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun1.mutagen.io"},
			},
			{
				URLs: []string{"stun:stun2.mutagen.io"},
			},
		},
	})
	if err != nil {
		return nil, nil, ErrorSeverityUnrecoverable, fmt.Errorf("unable to create new peer connection: %w", err)
	}

	// Track success. If we return without succeeding, then ensure that the
	// connection is closed out.
	var successful bool
	defer func() {
		if !successful {
			peerConnection.Close()
		}
	}()

	// Track connection state changes, dispatching connectivity and error states
	// appropriately.
	peerConnectionConnected := make(chan struct{})
	peerConnectionFailures := make(chan error, 1)
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		// Log the state change.
		c.logger.Info("Connection state change to", state)

		// Handle the case of a connected state. We treat repeated connection
		// events as errors.
		if state == webrtc.PeerConnectionStateConnected {
			select {
			case <-peerConnectionConnected:
				select {
				case peerConnectionFailures <- errors.New("repeated connection event"):
				default:
				}
			default:
				close(peerConnectionConnected)
			}
			return
		}

		// If an error state has occurred, then send a notification in a
		// non-blocking fashion (since we only care about the first error).
		var err error
		switch state {
		case webrtc.PeerConnectionStateDisconnected:
			err = errors.New("connection disconnected")
		case webrtc.PeerConnectionStateFailed:
			err = errors.New("connection failed")
		case webrtc.PeerConnectionStateClosed:
			err = errors.New("connection closed")
		}
		if err != nil {
			select {
			case peerConnectionFailures <- err:
			default:
			}
			return
		}
	})

	// TODO: We may also want to wire up SCTP transport errors to
	// peerConnectionFailures. Unfortunately the SCTP transport isn't accessible
	// directly from the peer connection (though oddly it is available on
	// individual data channels). It seems like this may just be an oversight.
	// See the following for additional discussion:
	// https://github.com/pion/webrtc/issues/754
	// https://github.com/pion/webrtc/commit/896f8e360f96498092b41ae24876de4ba012f63d

	// Initiate a client-side offer exchange.
	exchangeID, remoteOffer, remoteSignature, err := mutagenio.TunnelClientExchangeStart(
		ctx,
		c.tunnel.Identifier,
		c.tunnel.Token,
	)
	if err != nil {
		if err == mutagenio.ErrUnauthorized {
			return nil, nil, ErrorSeverityUnrecoverable, errors.New("invalid tunnel credentials")
		}
		return nil, nil, ErrorSeverityDelayedRecoverable, fmt.Errorf("unable to initiate offer exchange: %w", err)
	}

	// Decode the remote offer and signature.
	remoteOfferBytes, err := encoding.DecodeBase64(remoteOffer)
	if err != nil {
		return nil, nil, ErrorSeverityRecoverable, fmt.Errorf("unable to decode remote offer: %w", err)
	}
	remoteSignatureBytes, err := encoding.DecodeBase64(remoteSignature)
	if err != nil {
		return nil, nil, ErrorSeverityRecoverable, fmt.Errorf("unable to decode remote signature: %w", err)
	}

	// Verify the remote offer signature. We treat this as an unrecoverable
	// error since it's an indication of a man-in-the-middle attack.
	signatureMatch := verifyOfferSignature(
		remoteOfferBytes,
		c.tunnel.Version.hmacHash(),
		c.tunnel.Secret,
		remoteSignatureBytes,
	)
	if !signatureMatch {
		return nil, nil, ErrorSeverityUnrecoverable, errors.New("remote offer has incorrect signature")
	}

	// Unmarshal the remote session description.
	remoteSessionDescription := webrtc.SessionDescription{}
	if err := json.Unmarshal(remoteOfferBytes, &remoteSessionDescription); err != nil {
		return nil, nil, ErrorSeverityRecoverable, fmt.Errorf("unable to unmarshal remote session description: %w", err)
	}

	// Set the remote session description.
	if err := peerConnection.SetRemoteDescription(remoteSessionDescription); err != nil {
		return nil, nil, ErrorSeverityRecoverable, fmt.Errorf("unable to set remote session description: %w", err)
	}

	// Compute the local session description.
	localSessionDescription, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return nil, nil, ErrorSeverityRecoverable, fmt.Errorf("unable to create local session description: %w", err)
	}

	// Set the local session description.
	if err := peerConnection.SetLocalDescription(localSessionDescription); err != nil {
		return nil, nil, ErrorSeverityRecoverable, fmt.Errorf("unable to set local session description: %w", err)
	}

	// Marshal the local session description.
	localOfferBytes, err := json.Marshal(localSessionDescription)
	if err != nil {
		return nil, nil, ErrorSeverityRecoverable, fmt.Errorf("unable to marshal local session description: %w", err)
	}

	// Compute the local offer signature.
	localSignatureBytes := signOffer(
		localOfferBytes,
		c.tunnel.Version.hmacHash(),
		c.tunnel.Secret,
	)

	// Encode the local offer and signature.
	localOffer := encoding.EncodeBase64(localOfferBytes)
	localSignature := encoding.EncodeBase64(localSignatureBytes)

	// Finalize the client-side offer exchange.
	err = mutagenio.TunnelClientExchangeFinish(
		ctx,
		c.tunnel.Identifier, c.tunnel.Token,
		exchangeID, localOffer, localSignature,
	)
	if err != nil {
		if err == mutagenio.ErrUnauthorized {
			return nil, nil, ErrorSeverityUnrecoverable, errors.New("invalid tunnel credentials")
		}
		return nil, nil, ErrorSeverityDelayedRecoverable, fmt.Errorf("unable to finalize offer exchange: %w", err)
	}

	// Wait for the connection to complete.
	select {
	case <-peerConnectionConnected:
	case err := <-peerConnectionFailures:
		return nil, nil, ErrorSeverityRecoverable, fmt.Errorf("peer connection failure: %w", err)
	}

	// Success.
	successful = true
	return peerConnection, peerConnectionFailures, ErrorSeverityRecoverable, nil
}

func (c *controller) serve(
	ctx context.Context,
	peerConnection *webrtc.PeerConnection,
	failures chan error,
) error {
	// Track data channel indices so we can create unique names.
	var dataChannelIndex int

	// Loop and serve until there's an error or cancellation.
	for {
		// Wait for the next dial request, failure, or cancellation.
		var dialRequest dialRequest
		select {
		case dialRequest = <-c.dialRequests:
		case err := <-failures:
			return fmt.Errorf("peer connection failure: %w", err)
		case <-ctx.Done():
			return errors.New("cancelled")
		}

		// Open a new data channel.
		dataChannelName := fmt.Sprintf("session-%d", dataChannelIndex)
		dataChannelIndex++
		dataChannel, err := peerConnection.CreateDataChannel(dataChannelName, nil)
		if err != nil {
			select {
			case dialRequest.errors <- err:
			case <-dialRequest.ctx.Done():
			}
			return fmt.Errorf("unable to create data channel: %w", err)
		}

		// Increment session counts and extract the current state object.
		c.stateLock.Lock()
		c.state.ActiveSessions += 1
		c.state.TotalSessions += 1
		state := c.state
		c.stateLock.Unlock()

		// Create a callback that will update the state when the associated
		// connection is closed. We use the state object that we extracted
		// earlier since the state object associated with the controller may be
		// different at closing time. The locks, however, will remain the same.
		closureCallback := func() {
			c.stateLock.Lock()
			state.ActiveSessions -= 1
			c.stateLock.Unlock()
		}

		// Wrap the channel in a connection.
		connection := webrtcutil.NewConnection(dataChannel, closureCallback)

		// Return the connection.
		select {
		case dialRequest.results <- connection:
		case <-dialRequest.ctx.Done():
			connection.Close()
		}
	}
}

// dial performs a dial operation by creating a new data channel, requesting
// that it be connected to an agent binary running on the remote endpoint in the
// specified mode, and performing an agent handshake operation.
func (c *controller) dial(ctx context.Context, mode string) (net.Conn, error) {
	// Submit the dial request and then wait for the dial response. For both
	// operations, we monitor for cancellation. It's imperative that, after
	// successful submission of the dialing request, this function only exit in
	// cases where the context is cancelled (or, equivalently, that the context
	// that's submitted in the request is cancelled when this function exits).
	// If this behavior isn't adhered to, then the serving loop will block
	// forever while trying to return the response. If, during a future
	// refactor, the need arises to return after submission of the dialing
	// request but before receiving a dial response, then the provided context
	// should just be wrapped in a cancellable subcontext before being submitted
	// in the dialing request, and the cancellation of this subcontext should be
	// deferred.
	dialRequest := dialRequest{
		ctx:     ctx,
		results: make(chan net.Conn),
		errors:  make(chan error),
	}
	var connection net.Conn
	select {
	case c.dialRequests <- dialRequest:
	case <-ctx.Done():
		return nil, errors.New("cancelled")
	}
	select {
	case connection = <-dialRequest.results:
	case err := <-dialRequest.errors:
		return nil, err
	case <-ctx.Done():
		return nil, errors.New("cancelled")
	}

	// Send an initialization request.
	initializeRequest := &InitializeRequestVersion1{
		VersionMajor: mutagen.VersionMajor,
		VersionMinor: mutagen.VersionMinor,
		VersionPatch: mutagen.VersionPatch,
		Mode:         mode,
	}
	if err := encoding.EncodeProtobuf(connection, initializeRequest); err != nil {
		connection.Close()
		return nil, fmt.Errorf("unable to send initialization request: %w", err)
	}

	// Receive an initialization response.
	initializeResponse := &InitializeResponseVersion1{}
	if err := encoding.DecodeProtobuf(connection, initializeResponse); err != nil {
		connection.Close()
		return nil, fmt.Errorf("unable to receive initialization response: %w", err)
	}

	// Perform an agent handshake.
	if err := agent.ClientHandshake(connection); err != nil {
		connection.Close()
		return nil, fmt.Errorf("agent handshake failure: %w", err)
	}

	// Perform a Mutagen version handshake.
	if err := mutagen.ClientVersionHandshake(connection); err != nil {
		connection.Close()
		return nil, fmt.Errorf("version handshake failure: %w", err)
	}

	// Success.
	return connection, nil
}
