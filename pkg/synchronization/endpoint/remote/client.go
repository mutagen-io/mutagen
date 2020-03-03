package remote

import (
	"context"
	"net"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/proto"

	"github.com/mutagen-io/mutagen/pkg/compression"
	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
)

// endpointClient provides an implementation of synchronization.Endpoint by
// acting as a proxy for a remotely hosted synchronization.Endpoint.
type endpointClient struct {
	// connection is the control stream connection.
	connection net.Conn
	// encoder is the control stream encoder.
	encoder *encoding.ProtobufEncoder
	// decoder is the control stream decoder.
	decoder *encoding.ProtobufDecoder
	// lastSnapshotBytes is the serialized form of the last snapshot received
	// from the remote endpoint.
	lastSnapshotBytes []byte
}

// NewEndpoint creates a new remote synchronization.Endpoint operating over the
// specified connection with the specified metadata. If this function fails,
// then the provided connection will be closed. Once the endpoint has been
// established, the underlying connection is owned by that endpoint and will be
// closed when the endpoint is shut down.
func NewEndpoint(
	connection net.Conn,
	root string,
	session string,
	version synchronization.Version,
	configuration *synchronization.Configuration,
	alpha bool,
) (synchronization.Endpoint, error) {
	// Set up deferred closure of the connection if initialization fails.
	var successful bool
	defer func() {
		if !successful {
			connection.Close()
		}
	}()

	// Enable read/write compression on the connection.
	reader := compression.NewDecompressingReader(connection)
	writer := compression.NewCompressingWriter(connection)

	// Create an encoder and decoder.
	encoder := encoding.NewProtobufEncoder(writer)
	decoder := encoding.NewProtobufDecoder(reader)

	// Create and send the initialize request.
	request := &InitializeSynchronizationRequest{
		Root:          root,
		Session:       session,
		Version:       version,
		Configuration: configuration,
		Alpha:         alpha,
	}
	if err := encoder.Encode(request); err != nil {
		return nil, errors.Wrap(err, "unable to send initialize request")
	}

	// Receive the response and check for remote errors.
	response := &InitializeSynchronizationResponse{}
	if err := decoder.Decode(response); err != nil {
		return nil, errors.Wrap(err, "unable to receive transition response")
	} else if err = response.ensureValid(); err != nil {
		return nil, errors.Wrap(err, "invalid initialize response")
	} else if response.Error != "" {
		return nil, errors.Errorf("remote error: %s", response.Error)
	}

	// Success.
	successful = true
	return &endpointClient{
		connection: connection,
		encoder:    encoder,
		decoder:    decoder,
	}, nil
}

// Poll implements the Poll method for remote endpoints.
func (e *endpointClient) Poll(ctx context.Context) error {
	// Create and send the poll request.
	request := &EndpointRequest{Poll: &PollRequest{}}
	if err := e.encoder.Encode(request); err != nil {
		return errors.Wrap(err, "unable to send poll request")
	}

	// Create a subcontext that we can cancel to regulate transmission of the
	// completion request.
	completionCtx, cancel := context.WithCancel(ctx)

	// Create a Goroutine that will send a poll completion request when the
	// subcontext is cancelled.
	completionSendErrors := make(chan error, 1)
	go func() {
		<-completionCtx.Done()
		if err := e.encoder.Encode(&PollCompletionRequest{}); err != nil {
			completionSendErrors <- errors.Wrap(err, "unable to send completion request")
		} else {
			completionSendErrors <- nil
		}
	}()

	// Create a Goroutine that will receive a poll response.
	response := &PollResponse{}
	responseReceiveErrors := make(chan error, 1)
	go func() {
		if err := e.decoder.Decode(response); err != nil {
			responseReceiveErrors <- errors.Wrap(err, "unable to receive poll response")
		} else if err = response.ensureValid(); err != nil {
			responseReceiveErrors <- errors.Wrap(err, "invalid poll response")
		} else {
			responseReceiveErrors <- nil
		}
	}()

	// Wait for both a completion request to be sent and a response to be
	// received. Both of these will occur, though their order is not known. If
	// the completion request is sent first, then we know that the polling
	// context has been cancelled and that a response is on its way. In this
	// case, we still cancel the subcontext we created as required by the
	// context package to avoid leaking resources. If the response comes first,
	// then we need to force sending of the completion request and wait for the
	// result of that operation.
	var completionSendErr, responseReceiveErr error
	select {
	case completionSendErr = <-completionSendErrors:
		cancel()
		responseReceiveErr = <-responseReceiveErrors
	case responseReceiveErr = <-responseReceiveErrors:
		cancel()
		completionSendErr = <-completionSendErrors
	}

	// Check for transmission errors.
	if responseReceiveErr != nil {
		return responseReceiveErr
	} else if completionSendErr != nil {
		return completionSendErr
	}

	// Check for remote errors.
	if response.Error != "" {
		return errors.Errorf("remote error: %s", response.Error)
	}

	// Done.
	return nil
}

// Scan implements the Scan method for remote endpoints.
func (e *endpointClient) Scan(ctx context.Context, ancestor *core.Entry, full bool) (*core.Entry, bool, error, bool) {
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
		if err := buffer.Marshal(&core.Archive{Root: ancestor}); err != nil {
			return nil, false, errors.Wrap(err, "unable to marshal ancestor"), false
		}
		baseBytes = buffer.Bytes()
	}

	// Compute the base signature.
	baseSignature := engine.BytesSignature(baseBytes, 0)

	// Create and send the scan request.
	request := &EndpointRequest{
		Scan: &ScanRequest{
			BaseSnapshotSignature: baseSignature,
			Full:                  full,
		},
	}
	if err := e.encoder.Encode(request); err != nil {
		return nil, false, errors.Wrap(err, "unable to send scan request"), false
	}

	// Create a subcontext that we can cancel to regulate transmission of the
	// completion request.
	completionCtx, cancel := context.WithCancel(ctx)

	// Create a Goroutine that will send a scan completion request when the
	// subcontext is cancelled.
	completionSendErrors := make(chan error, 1)
	go func() {
		<-completionCtx.Done()
		if err := e.encoder.Encode(&ScanCompletionRequest{}); err != nil {
			completionSendErrors <- errors.Wrap(err, "unable to send completion request")
		} else {
			completionSendErrors <- nil
		}
	}()

	// Create a Goroutine that will receive a scan response.
	response := &ScanResponse{}
	responseReceiveErrors := make(chan error, 1)
	go func() {
		if err := e.decoder.Decode(response); err != nil {
			responseReceiveErrors <- errors.Wrap(err, "unable to receive scan response")
		} else if err = response.ensureValid(); err != nil {
			responseReceiveErrors <- errors.Wrap(err, "invalid scan response")
		} else {
			responseReceiveErrors <- nil
		}
	}()

	// Wait for both a completion request to be sent and a response to be
	// received. Both of these will occur, though their order is not known. If
	// the completion request is sent first, then we know that the scanning
	// context has been cancelled and that a response is on its way. In this
	// case, we still cancel the subcontext we created as required by the
	// context package to avoid leaking resources. If the response comes first,
	// then we need to force sending of the completion request and wait for the
	// result of that operation.
	var completionSendErr, responseReceiveErr error
	select {
	case completionSendErr = <-completionSendErrors:
		cancel()
		responseReceiveErr = <-responseReceiveErrors
	case responseReceiveErr = <-responseReceiveErrors:
		cancel()
		completionSendErr = <-completionSendErrors
	}

	// Check for transmission errors.
	if responseReceiveErr != nil {
		return nil, false, responseReceiveErr, false
	} else if completionSendErr != nil {
		return nil, false, completionSendErr, false
	}

	// Check for remote errors.
	if response.Error != "" {
		return nil, false, errors.Errorf("remote error: %s", response.Error), response.TryAgain
	}

	// Apply the remote's deltas to the expected snapshot.
	snapshotBytes, err := engine.PatchBytes(baseBytes, baseSignature, response.SnapshotDelta)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to patch base snapshot"), false
	}

	// Unmarshal the snapshot.
	archive := &core.Archive{}
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
func (e *endpointClient) Stage(paths []string, digests [][]byte) ([]string, []*rsync.Signature, rsync.Receiver, error) {
	// If there are no entries to stage, then we're done. We enforce (in message
	// validation) that stage requests aren't sent to the server with no entries
	// present.
	if len(paths) == 0 {
		return nil, nil, nil, nil
	}

	// Create and send the stage request.
	request := &EndpointRequest{
		Stage: &StageRequest{
			Paths:   paths,
			Digests: digests,
		},
	}
	if err := e.encoder.Encode(request); err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to send stage request")
	}

	// Receive the response and check for remote errors.
	response := &StageResponse{}
	if err := e.decoder.Decode(response); err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to receive stage response")
	} else if err = response.ensureValid(); err != nil {
		return nil, nil, nil, errors.Wrap(err, "invalid scan response")
	} else if response.Error != "" {
		return nil, nil, nil, errors.Errorf("remote error: %s", response.Error)
	}

	// If everything was already staged, then we can abort the staging
	// operation.
	if len(response.Paths) == 0 {
		return nil, nil, nil, nil
	}

	// Create an encoding receiver that can transmit rsync operations to the
	// remote.
	encoder := newProtobufRsyncEncoder(e.encoder)
	receiver := rsync.NewEncodingReceiver(encoder)

	// Success.
	return response.Paths, response.Signatures, receiver, nil
}

// Supply implements the Supply method for remote endpoints.
func (e *endpointClient) Supply(paths []string, signatures []*rsync.Signature, receiver rsync.Receiver) error {
	// Create and send the supply request.
	request := &EndpointRequest{
		Supply: &SupplyRequest{
			Paths:      paths,
			Signatures: signatures,
		},
	}
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
	decoder := newProtobufRsyncDecoder(e.decoder)
	if err := rsync.DecodeToReceiver(decoder, uint64(len(paths)), receiver); err != nil {
		return errors.Wrap(err, "unable to decode and forward rsync operations")
	}

	// Success.
	return nil
}

// Transition implements the Transition method for remote endpoints.
func (e *endpointClient) Transition(ctx context.Context, transitions []*core.Change) ([]*core.Entry, []*core.Problem, bool, error) {
	// Create and send the transition request.
	request := &EndpointRequest{
		Transition: &TransitionRequest{
			Transitions: transitions,
		},
	}
	if err := e.encoder.Encode(request); err != nil {
		return nil, nil, false, errors.Wrap(err, "unable to send transition request")
	}

	// Create a subcontext that we can cancel to regulate transmission of the
	// completion request.
	completionCtx, cancel := context.WithCancel(ctx)

	// Create a Goroutine that will send a transition completion request when
	// the subcontext is cancelled.
	completionSendErrors := make(chan error, 1)
	go func() {
		<-completionCtx.Done()
		if err := e.encoder.Encode(&TransitionCompletionRequest{}); err != nil {
			completionSendErrors <- errors.Wrap(err, "unable to send completion request")
		} else {
			completionSendErrors <- nil
		}
	}()

	// Create a Goroutine that will receive a transition response.
	response := &TransitionResponse{}
	responseReceiveErrors := make(chan error, 1)
	go func() {
		if err := e.decoder.Decode(response); err != nil {
			responseReceiveErrors <- errors.Wrap(err, "unable to receive transition response")
		} else if err = response.ensureValid(len(transitions)); err != nil {
			responseReceiveErrors <- errors.Wrap(err, "invalid transition response")
		} else {
			responseReceiveErrors <- nil
		}
	}()

	// Wait for both a completion request to be sent and a response to be
	// received. Both of these will occur, though their order is not known. If
	// the completion request is sent first, then we know that the transition
	// context has been cancelled and that a response is on its way. In this
	// case, we still cancel the subcontext we created as required by the
	// context package to avoid leaking resources. If the response comes first,
	// then we need to force sending of the completion request and wait for the
	// result of that operation.
	var completionSendErr, responseReceiveErr error
	select {
	case completionSendErr = <-completionSendErrors:
		cancel()
		responseReceiveErr = <-responseReceiveErrors
	case responseReceiveErr = <-responseReceiveErrors:
		cancel()
		completionSendErr = <-completionSendErrors
	}

	// Check for transmission errors.
	if responseReceiveErr != nil {
		return nil, nil, false, responseReceiveErr
	} else if completionSendErr != nil {
		return nil, nil, false, completionSendErr
	}

	// Check for remote errors.
	if response.Error != "" {
		return nil, nil, false, errors.Errorf("remote error: %s", response.Error)
	}

	// HACK: Extract the wrapped results.
	results := make([]*core.Entry, len(response.Results))
	for r, result := range response.Results {
		results[r] = result.Root
	}

	// Success.
	return results, response.Problems, response.StagerMissingFiles, nil
}

// Shutdown implements the Shutdown method for remote endpoints.
func (e *endpointClient) Shutdown() error {
	// Close the underlying connection. This will cause all stream reads/writes
	// to unblock.
	return e.connection.Close()
}
