package session

import (
	contextpkg "context"
	"encoding/gob"
	"net"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// remoteEndpointClient is an endpoint implementation that provides a proxy for
// another endpoint over a network. It is designed to be paired with
// ServeEndpoint.
type remoteEndpointClient struct {
	// connection is the control stream connection.
	connection net.Conn
	// encoder is the control stream encoder.
	encoder *gob.Encoder
	// decoder is the control stream decoder.
	decoder *gob.Decoder
	// preservesExecutability indicates whether or not the remote endpoint
	// preserves executability.
	preservesExecutability bool
}

// newRemoteEndpoint constructs a new remote endpoint instance using the
// specified connection.
func newRemoteEndpoint(
	connection net.Conn,
	session string,
	version Version,
	root string,
	ignores []string,
	alpha bool,
) (endpoint, error) {
	// Create encoders and decoders.
	encoder := gob.NewEncoder(connection)
	decoder := gob.NewDecoder(connection)

	// Create and send the initialize request.
	request := initializeRequest{
		Session: session,
		Version: version,
		Root:    root,
		Ignores: ignores,
		Alpha:   alpha,
	}
	if err := encoder.Encode(request); err != nil {
		connection.Close()
		return nil, errors.Wrap(err, "unable to send initialize request")
	}

	// Receive the response and check for remote errors.
	var response initializeResponse
	if err := decoder.Decode(&response); err != nil {
		connection.Close()
		return nil, errors.Wrap(err, "unable to receive transition response")
	} else if response.Error != "" {
		connection.Close()
		return nil, errors.Errorf("remote error: %s", response.Error)
	}

	// Success.
	return &remoteEndpointClient{
		connection:             connection,
		encoder:                encoder,
		decoder:                decoder,
		preservesExecutability: response.PreservesExecutability,
	}, nil
}

func (e *remoteEndpointClient) poll(context contextpkg.Context) error {
	// Create and send the poll request.
	request := endpointRequest{Poll: &pollRequest{}}
	if err := e.encoder.Encode(request); err != nil {
		return errors.Wrap(err, "unable to send poll request")
	}

	// Wrap the completion context in a context that we can cancel in order to
	// force sending the completion response if we receive an event. The context
	// may be cancelled before we return (in the event that we receive an early
	// completion request), but we defer its (idempotent) cancellation to ensure
	// the context is cancelled.
	completionContext, forceCompletionSend := contextpkg.WithCancel(context)
	defer forceCompletionSend()

	// Create a Goroutine that will send a poll completion request when the
	// context is cancelled.
	completionSendResults := make(chan error, 1)
	go func() {
		<-completionContext.Done()
		completionSendResults <- errors.Wrap(
			e.encoder.Encode(pollCompletionRequest{}),
			"unable to send poll completion request",
		)
	}()

	// Create a Goroutine that will receive a poll response.
	responseReceiveResults := make(chan error, 1)
	go func() {
		var response pollResponse
		if err := e.decoder.Decode(&response); err != nil {
			responseReceiveResults <- errors.Wrap(err, "unable to receive poll response")
		} else if response.Error != "" {
			responseReceiveResults <- errors.Errorf("remote error: %s", response.Error)
		}
		responseReceiveResults <- nil
	}()

	// Wait for both a completion encode to finish and a response to be
	// received. Both of these will happen, though their order is not
	// guaranteed. If the completion send comes first, we know the response is
	// on its way. If the response comes first, we need to force the completion
	// send.
	var completionSendErr, responseReceiveErr error
	select {
	case completionSendErr = <-completionSendResults:
		responseReceiveErr = <-responseReceiveResults
	case responseReceiveErr = <-responseReceiveResults:
		forceCompletionSend()
		completionSendErr = <-completionSendResults
	}

	// Check for errors.
	if responseReceiveErr != nil {
		return responseReceiveErr
	} else if completionSendErr != nil {
		return completionSendErr
	}

	// Done.
	return nil
}

func (e *remoteEndpointClient) scan(ancestor *sync.Entry) (*sync.Entry, bool, error) {
	// Create an rsync engine.
	engine := rsync.NewEngine()

	// Marshal the ancestor and compute its rsync signature. We'll use it as a
	// base for an rsync transfer of the serialized snapshot.
	ancestorBytes, err := marshalEntry(ancestor, true)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to marshal ancestor")
	}
	ancestorSignature := engine.BytesSignature(ancestorBytes, 0)

	// Create and send the scan request.
	request := endpointRequest{Scan: &scanRequest{ancestorSignature}}
	if err := e.encoder.Encode(request); err != nil {
		return nil, false, errors.Wrap(err, "unable to send scan request")
	}

	// Receive the response.
	var response scanResponse
	if err := e.decoder.Decode(&response); err != nil {
		return nil, false, errors.Wrap(err, "unable to receive scan response")
	}

	// Check if the endpoint says we should try again.
	if response.TryAgain {
		return nil, true, nil
	}

	// Apply the remote's deltas to the expected snapshot.
	snapshotBytes, err := engine.PatchBytes(ancestorBytes, ancestorSignature, response.SnapshotDelta)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to patch base snapshot")
	}

	// Unmarshal the snapshot.
	snapshot, err := unmarshalEntry(snapshotBytes)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to unmarshal snapshot")
	}

	// Ensure that the snapshot is valid since it came over the network.
	if err = snapshot.EnsureValid(); err != nil {
		return nil, false, errors.Wrap(err, "invalid snapshot received")
	}

	// If the endpoint doesn't preserve executability, then propagate
	// executability from the ancestor.
	if !e.preservesExecutability {
		snapshot = sync.PropagateExecutability(ancestor, snapshot)
	}

	// Success.
	return snapshot, false, nil
}

func (e *remoteEndpointClient) stage(paths []string, entries []*sync.Entry) ([]string, []rsync.Signature, rsync.Receiver, error) {
	// Create and send the stage request.
	request := endpointRequest{Stage: &stageRequest{paths, entries}}
	if err := e.encoder.Encode(request); err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to send stage request")
	}

	// Receive the response and check for remote errors.
	var response stageResponse
	if err := e.decoder.Decode(&response); err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to receive stage response")
	} else if response.Error != "" {
		return nil, nil, nil, errors.Errorf("remote error: %s", response.Error)
	}

	// Create an encoding receiver that can transmit rsync operations to the
	// remote.
	receiver := rsync.NewEncodingReceiver(e.encoder)

	// Success.
	return response.Paths, response.Signatures, receiver, nil
}

func (e *remoteEndpointClient) supply(paths []string, signatures []rsync.Signature, receiver rsync.Receiver) error {
	// Create and send the supply request.
	request := endpointRequest{Supply: &supplyRequest{paths, signatures}}
	if err := e.encoder.Encode(request); err != nil {
		// TODO: Should we find a way to finalize the receiver here? That's a
		// private rsync method, and there shouldn't be any resources in the
		// receiver in need of finalizing here, but it would be worth thinking
		// about for consistency.
		return errors.Wrap(err, "unable to send supply request")
	}

	// We don't receive a response to ensure that the remote is ready to
	// transmit, because there aren't really any errors that we can detect
	// before transmission starts and there's no way to transmit them once
	// transmission starts. If DecodeToReceiver succeeds, we can assume that the
	// forwarding succeeded, and if it fails, there's really no way for us to
	// get error information from the remote.

	// The endpoint should now forward rsync operations, so we need to decode
	// and forward them to the receiver. If this operation completes
	// successfully, supplying is complete and successful.
	if err := rsync.DecodeToReceiver(e.decoder, uint64(len(paths)), receiver); err != nil {
		return errors.Wrap(err, "unable to decode and forward rsync operations")
	}

	// Success.
	return nil
}

func (e *remoteEndpointClient) transition(transitions []*sync.Change) ([]*sync.Change, []*sync.Problem, error) {
	// Create and send the transition request.
	request := endpointRequest{Transition: &transitionRequest{transitions}}
	if err := e.encoder.Encode(request); err != nil {
		return nil, nil, errors.Wrap(err, "unable to send transition request")
	}

	// Receive the response and check for remote errors.
	var response transitionResponse
	if err := e.decoder.Decode(&response); err != nil {
		return nil, nil, errors.Wrap(err, "unable to receive transition response")
	} else if response.Error != "" {
		return nil, nil, errors.Errorf("remote error: %s", response.Error)
	}

	// Validate the response internals since they came over the wire.
	for _, c := range response.Changes {
		if err := c.EnsureValid(); err != nil {
			return nil, nil, errors.Wrap(err, "received invalid change")
		}
	}
	for _, p := range response.Problems {
		if err := p.EnsureValid(); err != nil {
			return nil, nil, errors.Wrap(err, "received invalid problem")
		}
	}

	// Success.
	return response.Changes, response.Problems, nil
}

func (e *remoteEndpointClient) shutdown() error {
	// Close the underlying connection. This will cause all stream reads/writes
	// to unblock.
	return e.connection.Close()
}
