package agent

import (
	contextpkg "context"
	"encoding/gob"
	"net"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/proto"

	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// endpointClient provides an implementation of session.Endpoint by acting as a
// proxy for a remotely hosted session.Endpoint.
type endpointClient struct {
	// connection is the control stream connection.
	connection net.Conn
	// encoder is the control stream encoder.
	encoder *gob.Encoder
	// decoder is the control stream decoder.
	decoder *gob.Decoder
	// lastSnapshotBytes is the serialized form of the last snapshot received
	// from the remote endpoint.
	lastSnapshotBytes []byte
}

// newEndpointClient constructs a new endpoint client instance using the
// specified connection and metadata.
func newEndpointClient(
	connection net.Conn,
	session string,
	version session.Version,
	root string,
	configuration *session.Configuration,
	alpha bool,
) (session.Endpoint, error) {
	// Wrap the connection in an encoder/decoder pair.
	encoder := gob.NewEncoder(connection)
	decoder := gob.NewDecoder(connection)

	// Create and send the initialize request.
	request := initializeRequest{
		Session:       session,
		Version:       version,
		Root:          root,
		Configuration: configuration,
		Alpha:         alpha,
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
	return &endpointClient{
		connection: connection,
		encoder:    encoder,
		decoder:    decoder,
	}, nil
}

// Poll implements the Poll method for remote endpoints.
func (e *endpointClient) Poll(context contextpkg.Context) error {
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

// Scan implements the Scan method for remote endpoints.
func (e *endpointClient) Scan(ancestor *sync.Entry) (*sync.Entry, bool, error, bool) {
	// Create an rsync engine.
	engine := rsync.NewEngine()

	// Compute the bytes that we'll use as the base for receiving the snapshot.
	// If we have the bytes from the last received snapshot, use those, because
	// they'll be more acccurate, but otherwise use the provided ancestor.
	var baseBytes []byte
	if e.lastSnapshotBytes != nil {
		baseBytes = e.lastSnapshotBytes
	} else {
		buffer := proto.NewBuffer(nil)
		buffer.SetDeterministic(true)
		if err := buffer.Marshal(&sync.Archive{Root: ancestor}); err != nil {
			return nil, false, errors.Wrap(err, "unable to marshal ancestor"), false
		}
		baseBytes = buffer.Bytes()
	}

	// Compute the base signature.
	baseSignature := engine.BytesSignature(baseBytes, 0)

	// Create and send the scan request.
	request := endpointRequest{Scan: &scanRequest{baseSignature}}
	if err := e.encoder.Encode(request); err != nil {
		return nil, false, errors.Wrap(err, "unable to send scan request"), false
	}

	// Receive the response.
	var response scanResponse
	if err := e.decoder.Decode(&response); err != nil {
		return nil, false, errors.Wrap(err, "unable to receive scan response"), false
	}

	// Check if the endpoint says we should try again.
	if response.TryAgain {
		return nil, false, errors.New(response.Error), true
	}

	// Apply the remote's deltas to the expected snapshot.
	snapshotBytes, err := engine.PatchBytes(baseBytes, baseSignature, response.SnapshotDelta)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to patch base snapshot"), false
	}

	// Unmarshal the snapshot.
	archive := &sync.Archive{}
	if err := proto.Unmarshal(snapshotBytes, archive); err != nil {
		return nil, false, errors.Wrap(err, "unable to unmarshal snapshot"), false
	}
	snapshot := archive.Root

	// Ensure that the snapshot is valid since it came over the network.
	if err = snapshot.EnsureValid(); err != nil {
		return nil, false, errors.Wrap(err, "invalid snapshot received"), false
	}

	// Store the bytes that gave us a successful snapshot.
	e.lastSnapshotBytes = snapshotBytes

	// Success.
	return snapshot, response.PreservesExecutability, nil, false
}

// Stage implements the Stage method for remote endpoints.
func (e *endpointClient) Stage(entries map[string][]byte) ([]string, []*rsync.Signature, rsync.Receiver, error) {
	// Create and send the stage request.
	request := endpointRequest{Stage: &stageRequest{entries}}
	if err := e.encoder.Encode(request); err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to send stage request")
	}

	// Receive the response and check for remote errors.
	var response stageResponse
	if err := e.decoder.Decode(&response); err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to receive stage response")
	} else if response.Error != "" {
		return nil, nil, nil, errors.Errorf("remote error: %s", response.Error)
	} else if len(response.Paths) != len(response.Signatures) {
		return nil, nil, nil, errors.New("number of signatures returned does not match number of paths")
	}

	// If everything was already staged, then we can abort the staging
	// operation.
	if len(response.Paths) == 0 {
		return nil, nil, nil, nil
	}

	// Create an encoding receiver that can transmit rsync operations to the
	// remote.
	encoder := func(t *rsync.Transmission) error {
		return e.encoder.Encode(t)
	}
	receiver := rsync.NewEncodingReceiver(encoder)

	// Success.
	return response.Paths, response.Signatures, receiver, nil
}

// Supply implements the Supply method for remote endpoints.
func (e *endpointClient) Supply(paths []string, signatures []*rsync.Signature, receiver rsync.Receiver) error {
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
	decoder := func(t *rsync.Transmission) error {
		return e.decoder.Decode(t)
	}
	if err := rsync.DecodeToReceiver(decoder, uint64(len(paths)), receiver); err != nil {
		return errors.Wrap(err, "unable to decode and forward rsync operations")
	}

	// Success.
	return nil
}

// Transition implements the Transition method for remote endpoints.
func (e *endpointClient) Transition(transitions []*sync.Change) ([]*sync.Entry, []*sync.Problem, error) {
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

	// HACK: Extract the wrapped results.
	results := make([]*sync.Entry, len(response.Results))
	for r, result := range response.Results {
		if result == nil {
			return nil, nil, errors.New("nil result wrapper received")
		}
		results[r] = result.Root
	}

	// Validate the response internals since they came over the wire.
	if len(results) != len(transitions) {
		return nil, nil, errors.New("transition results have invalid length")
	}
	for _, e := range results {
		if err := e.EnsureValid(); err != nil {
			return nil, nil, errors.Wrap(err, "received invalid entry")
		}
	}
	for _, p := range response.Problems {
		if err := p.EnsureValid(); err != nil {
			return nil, nil, errors.Wrap(err, "received invalid problem")
		}
	}

	// Success.
	return results, response.Problems, nil
}

// Shutdown implements the Shutdown method for remote endpoints.
func (e *endpointClient) Shutdown() error {
	// Close the underlying connection. This will cause all stream reads/writes
	// to unblock.
	return e.connection.Close()
}
