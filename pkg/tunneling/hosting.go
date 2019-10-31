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
	hostParameters *TunnelHostParameters,
) (ErrorSeverity, error) {
	// Create an unconnected peer connection and defer its closure.
	// TODO: Switch to Mutagen STUN servers. Potentially load from API.
	// TODO: Add credentials to these servers if supporting TURN.
	peerConnection, err := webrtcutil.API.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		return ErrorSeverityUnrecoverable, fmt.Errorf("unable to create new peer connection: %w", err)
	}
	defer peerConnection.Close()

	// Track any states that indicate failure for the connection. If any are
	// detected, then the tracking channel will be populated.
	peerConnectionFailures := make(chan struct{}, 1)
	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		logger.Println("Connection state change to", state)
		failed := state == webrtc.PeerConnectionStateDisconnected ||
			state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed
		if failed {
			select {
			case peerConnectionFailures <- struct{}{}:
			default:
			}
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
		hostParameters.Version.hmacHash(),
		hostParameters.Secret,
	)

	// Encode the local offer and signature.
	localOffer := encoding.EncodeBase64(localOfferBytes)
	localSignature := encoding.EncodeBase64(localSignatureBytes)

	// Perform a host-side offer exchange.
	remoteOffer, remoteSignature, err := mutagenio.TunnelHostExchange(
		ctx,
		hostParameters.Identifier,
		hostParameters.Token,
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
		hostParameters.Version.hmacHash(),
		hostParameters.Secret,
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

	// Create a context to regulate hosting.
	hostingCtx, hostingCancel := context.WithCancel(ctx)
	defer hostingCancel()

	// Track incoming data channels.
	dataChannels := make(chan *webrtc.DataChannel)
	peerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		logger.Println("Received data channel:", dataChannel.Label())
		select {
		case dataChannels <- dataChannel:
		case <-hostingCtx.Done():
		}
	})

	// Loop indefinitely, watching for incoming data channels, failure, and
	// cancellation.
	for {
		select {
		case dataChannel := <-dataChannels:
			go hostDataChannel(
				hostingCtx,
				logger.Sublogger(dataChannel.Label()),
				dataChannel,
				hostParameters.Version,
			)
		case <-peerConnectionFailures:
			// TODO: Can we get more detailed failure information?
			return ErrorSeverityRecoverable, errors.New("peer connection failure")
		case <-hostingCtx.Done():
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
	// Convert the data channel to a connection and ensure it's closed when
	// we're done.
	connection, err := webrtcutil.NewConnection(dataChannel, nil)
	if err != nil {
		logger.Println("Unable to create data channel connection:", err)
		dataChannel.Close()
		return
	}
	defer connection.Close()

	// Create a utility function to send an initialization response.
	sendInitializationResponse := func(err error) error {
		var errorMessage string
		if err != nil {
			logger.Println("Initialization failed with error:", err)
			errorMessage = err.Error()
		} else {
			logger.Println("Initialization succeeded")
		}
		return encoding.EncodeProtobuf(connection, &InitializeResponseVersion1{
			Error: errorMessage,
		})
	}

	// Receive an initialization request.
	initializeRequest := &InitializeRequestVersion1{}
	if err := encoding.DecodeProtobuf(connection, initializeRequest); err != nil {
		logger.Println("Unable to decode initialization request:", err)
		return
	} else if err = initializeRequest.ensureValid(); err != nil {
		sendInitializationResponse(fmt.Errorf("Invalid initialization request: %w", err))
		return
	}

	// Ensure that a user has not been specified.
	// TODO: If we do eventually support user specification, then use the
	// principal package to parse the provided identifier.
	if initializeRequest.User != "" {
		sendInitializationResponse(errors.New("user specification not supported"))
		return
	}

	// Track our success in finding an agent binary.
	var agentPath string

	// If a directory of tunnel agents has been specified, then see if there's a
	// compatible agent binary.
	if tunnelAgentsPath := os.Getenv("MUTAGEN_TUNNEL_AGENTS"); tunnelAgentsPath != "" {
		agentPath = filepath.Join(
			tunnelAgentsPath,
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

	// If we didn't find an agent in the directory, then check if we're a
	// compatible version. If so, extract a temporary agent binary for the
	// current platform and defer its removal.
	// TODO: If we do eventually support user specification, then we need to
	// ensure that the binary we extract here, if any, is accessible to the user
	// that will be executing it. This will likely involve a change of ownership
	// and possibly a change of permission bits.
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
	agent.Stderr = logger.Sublogger("agent").Writer()
	if err := agent.Start(); err != nil {
		sendInitializationResponse(fmt.Errorf("unable to start agent: %w", err))
		return
	}

	// Send the initialization response.
	if err := sendInitializationResponse(nil); err != nil {
		logger.Println("Unable to send successful initialization response:", err)
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
		logger.Println("Connection forwarding failed with error:", err)
		connection.Close()
		return
	case <-ctx.Done():
		logger.Println("Cancelled")
		return
	}
}
