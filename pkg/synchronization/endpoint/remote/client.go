package remote

import (
	"bufio"
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/logging"
	streampkg "github.com/mutagen-io/mutagen/pkg/stream"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
)

// endpointClient provides an implementation of synchronization.Endpoint by
// acting as a proxy for a remotely hosted synchronization.Endpoint.
type endpointClient struct {
	// logger is the underlying logger.
	logger *logging.Logger
	// closer close the compression resources and the control stream.
	closer io.Closer
	// flusher flushes the outbound control stream.
	flusher streampkg.Flusher
	// encoder is the control stream encoder.
	encoder *encoding.ProtobufEncoder
	// decoder is the control stream decoder.
	decoder *encoding.ProtobufDecoder
	// lastSnapshotBytes is the serialized form of the last snapshot received
	// from the remote endpoint.
	lastSnapshotBytes []byte
}

// NewEndpoint creates a new remote synchronization.Endpoint operating over the
// specified stream with the specified metadata. If this function fails, then
// the provided stream will be closed. Once the endpoint has been established,
// the underlying stream is owned by the endpoint and will be closed when the
// endpoint is shut down. The provided stream must unblock read and write
// operations when closed.
func NewEndpoint(
	logger *logging.Logger,
	stream io.ReadWriteCloser,
	root string,
	session string,
	version synchronization.Version,
	configuration *synchronization.Configuration,
	alpha bool,
) (synchronization.Endpoint, error) {
	// Set up compression for the control stream.
	decompressor := flate.NewReader(bufio.NewReaderSize(stream, controlStreamBufferSize))
	outbound := bufio.NewWriterSize(stream, controlStreamBufferSize)
	compressor, _ := flate.NewWriter(outbound, flate.DefaultCompression)
	flusher := streampkg.MultiFlusher(compressor, outbound)

	// Create a closer for the control stream and compression resources.
	closer := streampkg.MultiCloser(compressor, decompressor, stream)

	// Set up deferred closure of the control stream and compression resources
	// in the event that initialization fails.
	var successful bool
	defer func() {
		if !successful {
			closer.Close()
		}
	}()

	// Create an encoder and a decoder for Protocol Buffers messages. The
	// compressor already implements internal buffering, but the decompressor
	// requires additional buffering to implement io.ByteReader.
	encoder := encoding.NewProtobufEncoder(compressor)
	decoder := encoding.NewProtobufDecoder(bufio.NewReader(decompressor))

	// Create and send the initialize request.
	request := &InitializeSynchronizationRequest{
		Root:          root,
		Session:       session,
		Version:       version,
		Configuration: configuration,
		Alpha:         alpha,
	}
	if err := encoder.Encode(request); err != nil {
		return nil, fmt.Errorf("unable to encode initialize request: %w", err)
	} else if err = flusher.Flush(); err != nil {
		return nil, fmt.Errorf("unable to transmit initialize request: %w", err)
	}

	// Receive the response and check for remote errors.
	response := &InitializeSynchronizationResponse{}
	if err := decoder.Decode(response); err != nil {
		return nil, fmt.Errorf("unable to receive transition response: %w", err)
	} else if err = response.ensureValid(); err != nil {
		return nil, fmt.Errorf("invalid initialize response: %w", err)
	} else if response.Error != "" {
		return nil, fmt.Errorf("remote error: %s", response.Error)
	}

	// Success.
	successful = true
	return &endpointClient{
		logger:  logger,
		closer:  closer,
		flusher: flusher,
		encoder: encoder,
		decoder: decoder,
	}, nil
}

// encodeAndFlush encodes a Protocol Buffers message using the underlying
// encoder and then flushes the control stream.
func (c *endpointClient) encodeAndFlush(message proto.Message) error {
	if err := c.encoder.Encode(message); err != nil {
		return err
	} else if err = c.flusher.Flush(); err != nil {
		return fmt.Errorf("message transmission failed: %w", err)
	}
	return nil
}

// Poll implements the Poll method for remote endpoints.
func (c *endpointClient) Poll(ctx context.Context) error {
	// Create and send the poll request.
	request := &EndpointRequest{Poll: &PollRequest{}}
	if err := c.encodeAndFlush(request); err != nil {
		return fmt.Errorf("unable to send poll request: %w", err)
	}

	// Create a subcontext that we can cancel to regulate transmission of the
	// completion request.
	completionCtx, cancel := context.WithCancel(ctx)

	// Create a Goroutine that will send a poll completion request when the
	// subcontext is cancelled.
	completionSendErrors := make(chan error, 1)
	go func() {
		<-completionCtx.Done()
		if err := c.encodeAndFlush(&PollCompletionRequest{}); err != nil {
			completionSendErrors <- fmt.Errorf("unable to send completion request: %w", err)
		} else {
			completionSendErrors <- nil
		}
	}()

	// Create a Goroutine that will receive a poll response.
	response := &PollResponse{}
	responseReceiveErrors := make(chan error, 1)
	go func() {
		if err := c.decoder.Decode(response); err != nil {
			responseReceiveErrors <- fmt.Errorf("unable to receive poll response: %w", err)
		} else if err = response.ensureValid(); err != nil {
			responseReceiveErrors <- fmt.Errorf("invalid poll response: %w", err)
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
		return fmt.Errorf("remote error: %s", response.Error)
	}

	// Done.
	return nil
}

// Scan implements the Scan method for remote endpoints.
func (c *endpointClient) Scan(ctx context.Context, ancestor *core.Entry, full bool) (*core.Snapshot, error, bool) {
	// Create an rsync engine.
	engine := rsync.NewEngine()

	// Compute the bytes that we'll use as the base for receiving the snapshot.
	// If we have the bytes from the last received snapshot, then use those,
	// because they'll be more acccurate, but otherwise use the provided
	// ancestor (with some probabilistic assumptions about filesystem behavior).
	var baselineBytes []byte
	if c.lastSnapshotBytes != nil {
		c.logger.Debug("Using last snapshot bytes as baseline")
		baselineBytes = c.lastSnapshotBytes
	} else {
		c.logger.Debug("Using ancestor-based snapshot as baseline")
		var err error
		marshaling := proto.MarshalOptions{Deterministic: true}
		baselineBytes, err = marshaling.Marshal(&core.Snapshot{
			Content:                ancestor,
			PreservesExecutability: true,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to marshal ancestor-based snapshot: %w", err), false
		}
	}

	// Compute the base signature.
	baselineSignature := engine.BytesSignature(baselineBytes, 0)

	// Create and send the scan request.
	request := &EndpointRequest{
		Scan: &ScanRequest{
			BaselineSnapshotSignature: baselineSignature,
			Full:                      full,
		},
	}
	if err := c.encodeAndFlush(request); err != nil {
		return nil, fmt.Errorf("unable to send scan request: %w", err), false
	}

	// Create a subcontext that we can cancel to regulate transmission of the
	// completion request.
	completionCtx, cancel := context.WithCancel(ctx)

	// Create a Goroutine that will send a scan completion request when the
	// subcontext is cancelled.
	completionSendErrors := make(chan error, 1)
	go func() {
		<-completionCtx.Done()
		if err := c.encodeAndFlush(&ScanCompletionRequest{}); err != nil {
			completionSendErrors <- fmt.Errorf("unable to send completion request: %w", err)
		} else {
			completionSendErrors <- nil
		}
	}()

	// Create a Goroutine that will receive a scan response.
	response := &ScanResponse{}
	responseReceiveErrors := make(chan error, 1)
	go func() {
		if err := c.decoder.Decode(response); err != nil {
			responseReceiveErrors <- fmt.Errorf("unable to receive scan response: %w", err)
		} else if err = response.ensureValid(); err != nil {
			responseReceiveErrors <- fmt.Errorf("invalid scan response: %w", err)
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
		return nil, responseReceiveErr, false
	} else if completionSendErr != nil {
		return nil, completionSendErr, false
	}

	// Check for remote errors.
	if response.Error != "" {
		return nil, fmt.Errorf("remote error: %s", response.Error), response.TryAgain
	}

	// Apply the remote's deltas to the expected snapshot.
	snapshotBytes, err := engine.PatchBytes(baselineBytes, baselineSignature, response.SnapshotDelta)
	if err != nil {
		return nil, fmt.Errorf("unable to patch base snapshot: %w", err), false
	}

	// If logging is enabled, then compute snapshot transmission statistics.
	if logging.CurrentLevel() >= logging.LevelDebug {
		var dataOperations, totalDataSize, blockOperations int
		for _, operation := range response.SnapshotDelta {
			if dataSize := len(operation.Data); dataSize > 0 {
				dataOperations++
				totalDataSize += dataSize
			} else {
				blockOperations++
			}
		}
		c.logger.Debugf("Snapshot delta yielded %d bytes using %d block operation(s) and %d data operation(s) totaling %d byte(s)",
			len(snapshotBytes), blockOperations, dataOperations, totalDataSize,
		)
	}

	// Unmarshal the snapshot.
	snapshot := &core.Snapshot{}
	if err := proto.Unmarshal(snapshotBytes, snapshot); err != nil {
		return nil, fmt.Errorf("unable to unmarshal snapshot: %w", err), false
	}

	// Ensure that the snapshot is valid since it came over the network. Ideally
	// we'd want this validation to be performed by the ensureValid method of
	// ScanResponse, but because this method requires rsync-based patching and
	// Protocol Buffers decoding before it actually has the underlying response,
	// we can't perform this validation into ScanResponse.ensureValid.
	if err = snapshot.EnsureValid(); err != nil {
		return nil, fmt.Errorf("invalid snapshot received: %w", err), false
	}

	// Store the bytes that gave us a successful snapshot so that we can use
	// them as a baseline for receiving the next snapshot, but only do this if
	// the snapshot content was non-nil (i.e. there were entries on disk). If we
	// received a snapshot with no entries, then chances are that it's coming
	// from a remote endpoint that hasn't yet been populated by content, meaning
	// its next transmission (after being populated) is going to be far closer
	// to ancestor than to the empty snapshot that it just sent, and thus we'll
	// want to use the serialized ancestor snapshot as the baseline until we
	// receive a populated snapshot.
	if snapshot.Content != nil {
		c.lastSnapshotBytes = snapshotBytes
	}

	// Success.
	return snapshot, nil, false
}

// Stage implements the Stage method for remote endpoints.
func (c *endpointClient) Stage(paths []string, digests [][]byte) ([]string, []*rsync.Signature, rsync.Receiver, error) {
	// Validate argument lengths and bail if there's nothing to stage.
	if len(paths) != len(digests) {
		return nil, nil, nil, errors.New("path count does not match digest count")
	} else if len(paths) == 0 {
		return nil, nil, nil, nil
	}

	// Create and send the stage request.
	request := &EndpointRequest{
		Stage: &StageRequest{
			Paths:   paths,
			Digests: digests,
		},
	}
	if err := c.encodeAndFlush(request); err != nil {
		return nil, nil, nil, fmt.Errorf("unable to send stage request: %w", err)
	}

	// Receive the response and check for remote errors.
	response := &StageResponse{}
	if err := c.decoder.Decode(response); err != nil {
		return nil, nil, nil, fmt.Errorf("unable to receive stage response: %w", err)
	} else if err = response.ensureValid(paths); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid stage response: %w", err)
	} else if response.Error != "" {
		return nil, nil, nil, fmt.Errorf("remote error: %s", response.Error)
	}

	// Handle the shorthand mechanism used by the remote to indicate that all
	// paths are required.
	requiredPaths := response.Paths
	if len(response.Paths) == 0 && len(response.Signatures) > 0 {
		requiredPaths = paths
	}

	// If everything was already staged, then we can abort the staging
	// operation.
	if len(requiredPaths) == 0 {
		return nil, nil, nil, nil
	}

	// Create an encoding receiver that can transmit rsync operations to the
	// remote.
	encoder := &protobufRsyncEncoder{encoder: c.encoder, flusher: c.flusher}
	receiver := rsync.NewEncodingReceiver(encoder)

	// Success.
	return requiredPaths, response.Signatures, receiver, nil
}

// Supply implements the Supply method for remote endpoints.
func (c *endpointClient) Supply(paths []string, signatures []*rsync.Signature, receiver rsync.Receiver) error {
	// Create and send the supply request.
	request := &EndpointRequest{
		Supply: &SupplyRequest{
			Paths:      paths,
			Signatures: signatures,
		},
	}
	if err := c.encodeAndFlush(request); err != nil {
		// TODO: Should we find a way to finalize the receiver here? That's a
		// private rsync method, and there shouldn't be any resources in the
		// receiver in need of finalizing here, but it would be worth thinking
		// about for consistency.
		return fmt.Errorf("unable to send supply request: %w", err)
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
	decoder := &protobufRsyncDecoder{decoder: c.decoder}
	if err := rsync.DecodeToReceiver(decoder, uint64(len(paths)), receiver); err != nil {
		return fmt.Errorf("unable to decode and forward rsync operations: %w", err)
	}

	// Success.
	return nil
}

// Transition implements the Transition method for remote endpoints.
func (c *endpointClient) Transition(ctx context.Context, transitions []*core.Change) ([]*core.Entry, []*core.Problem, bool, error) {
	// Create and send the transition request.
	request := &EndpointRequest{
		Transition: &TransitionRequest{
			Transitions: transitions,
		},
	}
	if err := c.encodeAndFlush(request); err != nil {
		return nil, nil, false, fmt.Errorf("unable to send transition request: %w", err)
	}

	// Create a subcontext that we can cancel to regulate transmission of the
	// completion request.
	completionCtx, cancel := context.WithCancel(ctx)

	// Create a Goroutine that will send a transition completion request when
	// the subcontext is cancelled.
	completionSendErrors := make(chan error, 1)
	go func() {
		<-completionCtx.Done()
		if err := c.encodeAndFlush(&TransitionCompletionRequest{}); err != nil {
			completionSendErrors <- fmt.Errorf("unable to send completion request: %w", err)
		} else {
			completionSendErrors <- nil
		}
	}()

	// Create a Goroutine that will receive a transition response.
	response := &TransitionResponse{}
	responseReceiveErrors := make(chan error, 1)
	go func() {
		if err := c.decoder.Decode(response); err != nil {
			responseReceiveErrors <- fmt.Errorf("unable to receive transition response: %w", err)
		} else if err = response.ensureValid(len(transitions)); err != nil {
			responseReceiveErrors <- fmt.Errorf("invalid transition response: %w", err)
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
		return nil, nil, false, fmt.Errorf("remote error: %s", response.Error)
	}

	// HACK: Extract the wrapped results.
	results := make([]*core.Entry, len(response.Results))
	for r, result := range response.Results {
		results[r] = result.Content
	}

	// Success.
	return results, response.Problems, response.StagerMissingFiles, nil
}

// Shutdown implements the Shutdown method for remote endpoints.
func (c *endpointClient) Shutdown() error {
	// Close the compression resources and the control stream. This will cause
	// all control stream reads/writes to unblock.
	return c.closer.Close()
}
