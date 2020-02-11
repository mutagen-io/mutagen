package tunneling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pion/webrtc/v2"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/mutagenio"
	"github.com/mutagen-io/mutagen/pkg/process"
	"github.com/mutagen-io/mutagen/pkg/tunneling/webrtcutil"
)

const (
	// HostTunnelRetryDelayTime is the amount of time to wait before retrying a
	// HostTunnel call after experiencing a failure with
	// ErrorSeverityDelayedRecoverable.
	HostTunnelRetryDelayTime = 5 * time.Second
)

// HostTunnel performs tunnel hosting with the specified host parameters. It
// should be called in a loop to facilitate reconnection on failure. In addition
// to returning an error, it also returns a boolean indicating whether or not
// that failure is unrecoverable. If the error is unrecoverable, the hosting
// loop should be terminated.
func HostTunnel(
	ctx context.Context,
	logger *logging.Logger,
	hostCredentials *TunnelHostCredentials,
) (ErrorSeverity, error) {
	// Create a cancellable subcontext to regulate our hosting Goroutines and
	// defer its cancellation.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Load the WebRTC API.
	api, err := loadWebRTCAPI()
	if err != nil {
		return ErrorSeverityUnrecoverable, fmt.Errorf("unable to initialize tunnel network API: %w", err)
	}

	// Create an unconnected peer connection and defer its closure.
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
		return ErrorSeverityUnrecoverable, fmt.Errorf("unable to create new peer connection: %w", err)
	}
	defer peerConnection.Close()

	// Track connection failure states.
	peerConnectionFailures := make(chan error, 1)
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		// Log the state change.
		logger.Info("Connection state change to", state)

		// If an error state has occurred, then send a notification. We send the
		// the error in a non-blocking fashion, because we only need to monitor
		// for the first error and we don't have control over which states (and
		// how many) we'll see.
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
		}
	})

	// TODO: We may also want to wire up SCTP transport errors to
	// peerConnectionFailures. Unfortunately the SCTP transport isn't accessible
	// directly from the peer connection (though oddly it is available on
	// individual data channels). It seems like this may just be an oversight.
	// See the following for additional discussion:
	// https://github.com/pion/webrtc/issues/754
	// https://github.com/pion/webrtc/commit/896f8e360f96498092b41ae24876de4ba012f63d

	// Track incoming data channels.
	dataChannels := make(chan *webrtc.DataChannel)
	peerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		logger.Info("Received data channel:", dataChannel.Label())
		select {
		case dataChannels <- dataChannel:
		case <-ctx.Done():
			dataChannel.Close()
		}
	})

	// Create the local session description.
	localSessionDescription, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return ErrorSeverityRecoverable, fmt.Errorf("unable to create local session description: %w", err)
	}

	// Set the local session description.
	if err := peerConnection.SetLocalDescription(localSessionDescription); err != nil {
		return ErrorSeverityRecoverable, fmt.Errorf("unable to set local session description: %w", err)
	}

	// Marshal the local session description.
	localOfferBytes, err := json.Marshal(localSessionDescription)
	if err != nil {
		return ErrorSeverityRecoverable, fmt.Errorf("unable to marshal local session description: %w", err)
	}

	// Compute the local offer signature.
	localSignatureBytes := signOffer(
		localOfferBytes,
		hostCredentials.Version.hmacHash(),
		hostCredentials.Secret,
	)

	// Encode the local offer and signature.
	localOffer := encoding.EncodeBase64(localOfferBytes)
	localSignature := encoding.EncodeBase64(localSignatureBytes)

	// Perform a host-side offer exchange.
	remoteOffer, remoteSignature, err := mutagenio.TunnelHostExchange(
		ctx,
		hostCredentials.Identifier,
		hostCredentials.Token,
		localOffer,
		localSignature,
	)
	if err != nil {
		if err == mutagenio.ErrUnauthorized {
			return ErrorSeverityUnrecoverable, errors.New("invalid tunnel credentials")
		}
		return ErrorSeverityDelayedRecoverable, fmt.Errorf("unable to initiate offer exchange: %w", err)
	}

	// Decode the remote offer and signature.
	remoteOfferBytes, err := encoding.DecodeBase64(remoteOffer)
	if err != nil {
		return ErrorSeverityRecoverable, fmt.Errorf("unable to decode remote offer: %w", err)
	}
	remoteSignatureBytes, err := encoding.DecodeBase64(remoteSignature)
	if err != nil {
		return ErrorSeverityRecoverable, fmt.Errorf("unable to decode remote signature: %w", err)
	}

	// Verify the remote offer signature. We treat this as an unrecoverable
	// error since it's an indication of a man-in-the-middle attack.
	signatureMatch := verifyOfferSignature(
		remoteOfferBytes,
		hostCredentials.Version.hmacHash(),
		hostCredentials.Secret,
		remoteSignatureBytes,
	)
	if !signatureMatch {
		return ErrorSeverityUnrecoverable, errors.New("remote offer has incorrect signature")
	}

	// Unmarshal the remote session description.
	remoteSessionDescription := webrtc.SessionDescription{}
	if err := json.Unmarshal(remoteOfferBytes, &remoteSessionDescription); err != nil {
		return ErrorSeverityRecoverable, fmt.Errorf("unable to unmarshal remote session description: %w", err)
	}

	// Set the remote session description.
	if err := peerConnection.SetRemoteDescription(remoteSessionDescription); err != nil {
		return ErrorSeverityRecoverable, fmt.Errorf("unable to set remote session description: %w", err)
	}

	// Loop indefinitely, watching for incoming data channels, peer connection
	// failure, and cancellation.
	for {
		select {
		case dataChannel := <-dataChannels:
			go hostDataChannel(
				ctx,
				logger.Sublogger(dataChannel.Label()),
				dataChannel,
				hostCredentials.Version,
			)
		case err := <-peerConnectionFailures:
			return ErrorSeverityRecoverable, fmt.Errorf("peer connection failure: %w", err)
		case <-ctx.Done():
			return ErrorSeverityRecoverable, errors.New("cancelled")
		}
	}
}

// hostDataChannel hosts an individual data channel within a tunnel.
func hostDataChannel(
	ctx context.Context,
	logger *logging.Logger,
	dataChannel *webrtc.DataChannel,
	tunnelVersion Version,
) {
	// Convert the data channel to a connection and defer its closure.
	connection := webrtcutil.NewConnection(dataChannel, nil)
	defer connection.Close()

	// Create a utility function to send an initialization response.
	sendInitializationResponse := func(err error) error {
		var errorMessage string
		if err != nil {
			logger.Info("Initialization failed with error:", err)
			errorMessage = err.Error()
		} else {
			logger.Info("Initialization succeeded")
		}
		return encoding.EncodeProtobuf(connection, &InitializeResponseVersion1{
			Error: errorMessage,
		})
	}

	// Receive an initialization request.
	initializeRequest := &InitializeRequestVersion1{}
	if err := encoding.DecodeProtobuf(connection, initializeRequest); err != nil {
		logger.Info("Unable to decode initialization request:", err)
		return
	} else if err = initializeRequest.ensureValid(); err != nil {
		sendInitializationResponse(fmt.Errorf("Invalid initialization request: %w", err))
		return
	}

	// Track our success in finding an agent binary.
	var agentPath string

	// Start by looking in the libexec directory to tunnel agents.
	if libexecPath, err := filesystem.LibexecPath(); err == nil {
		agentPath = filepath.Join(
			libexecPath, "mutagen", "agents",
			fmt.Sprintf("%d.%d",
				initializeRequest.VersionMajor,
				initializeRequest.VersionMinor,
			),
			process.ExecutableName(agent.BaseName, runtime.GOOS),
		)
		if metadata, err := os.Lstat(agentPath); err != nil {
			agentPath = ""
		} else if metadata.Mode()&os.ModeType != 0 {
			agentPath = ""
		}
	}

	// If we didn't find an agent in the libexec directory, then check if we're
	// a compatible version. If so, extract a temporary agent binary for the
	// current platform and defer its removal.
	if agentPath == "" {
		compatible := initializeRequest.VersionMajor == mutagen.VersionMajor &&
			initializeRequest.VersionMinor == mutagen.VersionMinor
		if compatible {
			if a, err := agent.ExecutableForPlatform(runtime.GOOS, runtime.GOARCH, ""); err == nil {
				agentPath = a
				defer os.Remove(a)
			}
		}
	}

	// If we haven't found a compatible agent binary, we have to abort.
	if agentPath == "" {
		sendInitializationResponse(errors.New("unable to find compatible agent"))
		return
	}

	// Create a subcontext in which the agent process can run and defer its
	// cancellation.
	agentCtx, agentCancel := context.WithCancel(ctx)
	defer agentCancel()

	// Start the agent process.
	agent := exec.CommandContext(agentCtx, agentPath, initializeRequest.Mode)
	agentInput, err := agent.StdinPipe()
	if err != nil {
		sendInitializationResponse(fmt.Errorf("unable to redirect agent input: %w", err))
		return
	}
	agentOutput, err := agent.StdoutPipe()
	if err != nil {
		sendInitializationResponse(fmt.Errorf("unable to redirect agent output: %w", err))
		return
	}
	agent.Stderr = logger.Sublogger("agent").Writer(logging.LevelInfo)
	if err := agent.Start(); err != nil {
		sendInitializationResponse(fmt.Errorf("unable to start agent: %w", err))
		return
	}

	// Send the initialization response.
	if err := sendInitializationResponse(nil); err != nil {
		logger.Info("Unable to send successful initialization response:", err)
		return
	}

	// Forward the connection to the agent process.
	copyErrors := make(chan error, 2)
	go func() {
		_, err := io.Copy(agentInput, connection)
		copyErrors <- err
	}()
	go func() {
		_, err := io.Copy(connection, agentOutput)
		copyErrors <- err
	}()

	// Wait for cancellation or connectivity failure.
	select {
	case err = <-copyErrors:
		if err != nil {
			logger.Info("Connection forwarding failed with error:", err)
		} else {
			logger.Info("Connection closed")
		}
		connection.Close()
		return
	case <-ctx.Done():
		logger.Info("Cancelled")
		return
	}
}
