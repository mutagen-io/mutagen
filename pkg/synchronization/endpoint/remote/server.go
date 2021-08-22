package remote

import (
	"context"
	"errors"
	"fmt"
	"net"

	"google.golang.org/protobuf/proto"

	"github.com/mutagen-io/mutagen/pkg/compression"
	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
	"github.com/mutagen-io/mutagen/pkg/synchronization/endpoint/local"
	"github.com/mutagen-io/mutagen/pkg/synchronization/rsync"
)

// endpointServer wraps a local endpoint instances and dispatches requests to
// this endpoint from an endpoint client.
type endpointServer struct {
	// encoder is the control stream encoder.
	encoder *encoding.ProtobufEncoder
	// decoder is the control stream decoder.
	decoder *encoding.ProtobufDecoder
	// endpoint is the underlying local endpoint.
	endpoint synchronization.Endpoint
}

// ServeEndpoint creates and serves a endpoint server on the specified
// connection. It enforces that the provided connection is closed by the time
// this function returns, regardless of failure.
func ServeEndpoint(logger *logging.Logger, connection net.Conn) error {
	// Defer closure of the connection.
	defer connection.Close()

	// Enable read/write compression on the connection.
	reader := compression.NewDecompressingReader(connection)
	writer := compression.NewCompressingWriter(connection)

	// Create an encoder and decoder.
	encoder := encoding.NewProtobufEncoder(writer)
	decoder := encoding.NewProtobufDecoder(reader)

	// Receive the initialize request. If this fails, then send a failure
	// response (even though the pipe is probably broken) and abort.
	request := &InitializeSynchronizationRequest{}
	if err := decoder.Decode(request); err != nil {
		err = fmt.Errorf("unable to receive initialize request: %w", err)
		encoder.Encode(&InitializeSynchronizationResponse{Error: err.Error()})
		return err
	}

	// Ensure that the initialization request is valid.
	if err := request.ensureValid(); err != nil {
		err = fmt.Errorf("invalid initialize request: %w", err)
		encoder.Encode(&InitializeSynchronizationResponse{Error: err.Error()})
		return err
	}

	// Expand and normalize the root path.
	if r, err := filesystem.Normalize(request.Root); err != nil {
		err = fmt.Errorf("unable to normalize synchronization root: %w", err)
		encoder.Encode(&InitializeSynchronizationResponse{Error: err.Error()})
		return err
	} else {
		request.Root = r
	}

	// Create the underlying endpoint. If it fails to create, then send a
	// failure response and abort. If it succeeds, then defer its closure.
	endpoint, err := local.NewEndpoint(
		logger,
		request.Root,
		request.Session,
		request.Version,
		request.Configuration,
		request.Alpha,
	)
	if err != nil {
		err = fmt.Errorf("unable to create underlying endpoint: %w", err)
		encoder.Encode(&InitializeSynchronizationResponse{Error: err.Error()})
		return err
	}
	defer endpoint.Shutdown()

	// Send a successful initialize response.
	if err = encoder.Encode(&InitializeSynchronizationResponse{}); err != nil {
		return fmt.Errorf("unable to send initialize response: %w", err)
	}

	// Create the server.
	server := &endpointServer{
		endpoint: endpoint,
		encoder:  encoder,
		decoder:  decoder,
	}

	// Server until an error occurs.
	return server.serve()
}

// serve is the main request handling loop.
func (s *endpointServer) serve() error {
	// Keep a reusable endpoint request.
	request := &EndpointRequest{}

	// Receive and process control requests until there's an error.
	for {
		// Receive the next request.
		*request = EndpointRequest{}
		if err := s.decoder.Decode(request); err != nil {
			return fmt.Errorf("unable to receive request: %w", err)
		} else if err = request.ensureValid(); err != nil {
			return fmt.Errorf("invalid endpoint request: %w", err)
		}

		// Handle the request based on type.
		if request.Poll != nil {
			if err := s.servePoll(request.Poll); err != nil {
				return fmt.Errorf("unable to serve poll request: %w", err)
			}
		} else if request.Scan != nil {
			if err := s.serveScan(request.Scan); err != nil {
				return fmt.Errorf("unable to serve scan request: %w", err)
			}
		} else if request.Stage != nil {
			if err := s.serveStage(request.Stage); err != nil {
				return fmt.Errorf("unable to serve stage request: %w", err)
			}
		} else if request.Supply != nil {
			if err := s.serveSupply(request.Supply); err != nil {
				return fmt.Errorf("unable to serve supply request: %w", err)
			}
		} else if request.Transition != nil {
			if err := s.serveTransition(request.Transition); err != nil {
				return fmt.Errorf("unable to serve transition request: %w", err)
			}
		} else {
			// TODO: Should we panic here? The request validation already
			// ensures that one and only one message component is set, so we
			// should never hit this condition.
			return errors.New("invalid request")
		}
	}
}

// servePoll serves a poll request.
func (s *endpointServer) servePoll(request *PollRequest) error {
	// Ensure the request is valid.
	if err := request.ensureValid(); err != nil {
		return fmt.Errorf("invalid poll request: %w", err)
	}

	// Create a cancellable context for executing the poll.
	ctx, cancel := context.WithCancel(context.Background())

	// Start a Goroutine to watch for the completion request.
	completionReceiveErrors := make(chan error, 1)
	go func() {
		request := &PollCompletionRequest{}
		if err := s.decoder.Decode(request); err != nil {
			completionReceiveErrors <- fmt.Errorf("unable to receive completion request: %w", err)
		} else if err = request.ensureValid(); err != nil {
			completionReceiveErrors <- fmt.Errorf("received invalid completion request: %w", err)
		} else {
			completionReceiveErrors <- nil
		}
	}()

	// Start a Goroutine to execute the poll and send a response when done.
	responseSendErrors := make(chan error, 1)
	go func() {
		// Perform polling and set up the response.
		var response *PollResponse
		if err := s.endpoint.Poll(ctx); err != nil {
			response = &PollResponse{
				Error: err.Error(),
			}
		} else {
			response = &PollResponse{}
		}

		// Send te response.
		if err := s.encoder.Encode(response); err != nil {
			responseSendErrors <- fmt.Errorf("unable to transmit response: %w", err)
		} else {
			responseSendErrors <- nil
		}
	}()

	// Wait for both a completion request to be received and a response to be
	// sent. Both of these will occur, though their order is not known. If the
	// completion request is received first, then we cancel the subcontext to
	// preempt the scan and force transmission of a response. If the response is
	// sent first, then we know the completion request is on its way. In this
	// case, we still cancel the subcontext we created as required by the
	// context package to avoid leaking resources.
	var responseSendErr, completionReceiveErr error
	select {
	case completionReceiveErr = <-completionReceiveErrors:
		cancel()
		responseSendErr = <-responseSendErrors
	case responseSendErr = <-responseSendErrors:
		cancel()
		completionReceiveErr = <-completionReceiveErrors
	}

	// Check for errors.
	if responseSendErr != nil {
		return responseSendErr
	} else if completionReceiveErr != nil {
		return completionReceiveErr
	}

	// Success.
	return nil
}

// serveScan serves a scan request.
func (s *endpointServer) serveScan(request *ScanRequest) error {
	// Ensure the request is valid.
	if err := request.ensureValid(); err != nil {
		return fmt.Errorf("invalid scan request: %w", err)
	}

	// Create a cancellable context for executing the scan.
	ctx, cancel := context.WithCancel(context.Background())

	// Start a Goroutine to watch for the completion request.
	completionReceiveErrors := make(chan error, 1)
	go func() {
		request := &ScanCompletionRequest{}
		if err := s.decoder.Decode(request); err != nil {
			completionReceiveErrors <- fmt.Errorf("unable to receive completion request: %w", err)
		} else if err = request.ensureValid(); err != nil {
			completionReceiveErrors <- fmt.Errorf("received invalid completion request: %w", err)
		} else {
			completionReceiveErrors <- nil
		}
	}()

	// Start a Goroutine to execute the scan and send a response when done.
	responseSendErrors := make(chan error, 1)
	go func() {
		// Configure Protocol Buffers marshaling to be deterministic.
		marshaling := proto.MarshalOptions{Deterministic: true}

		// Create an rsync engine.
		engine := rsync.NewEngine()

		// Perform a scan and set up the response.
		var response *ScanResponse
		snapshot, preservesExecutability, err, tryAgain := s.endpoint.Scan(ctx, nil, request.Full)
		if err != nil {
			response = &ScanResponse{
				Error:    err.Error(),
				TryAgain: tryAgain,
			}
		} else if snapshotBytes, err := marshaling.Marshal(&core.Archive{Content: snapshot}); err != nil {
			response = &ScanResponse{
				Error: fmt.Errorf("unable to marshal snapshot: %w", err).Error(),
			}
		} else {
			delta := engine.DeltafyBytes(snapshotBytes, request.BaseSnapshotSignature, 0)
			response = &ScanResponse{
				SnapshotDelta:          delta,
				PreservesExecutability: preservesExecutability,
			}
		}

		// Send the response.
		if err := s.encoder.Encode(response); err != nil {
			responseSendErrors <- fmt.Errorf("unable to transmit response: %w", err)
		} else {
			responseSendErrors <- nil
		}
	}()

	// Wait for both a completion request to be received and a response to be
	// sent. Both of these will occur, though their order is not known. If the
	// completion request is received first, then we cancel the subcontext to
	// preempt the scan and force transmission of a response. If the response is
	// sent first, then we know the completion request is on its way. In this
	// case, we still cancel the subcontext we created as required by the
	// context package to avoid leaking resources.
	var responseSendErr, completionReceiveErr error
	select {
	case completionReceiveErr = <-completionReceiveErrors:
		cancel()
		responseSendErr = <-responseSendErrors
	case responseSendErr = <-responseSendErrors:
		cancel()
		completionReceiveErr = <-completionReceiveErrors
	}

	// Check for errors.
	if responseSendErr != nil {
		return responseSendErr
	} else if completionReceiveErr != nil {
		return completionReceiveErr
	}

	// Success.
	return nil
}

// serveStage serves a stage request.
func (s *endpointServer) serveStage(request *StageRequest) error {
	// Ensure the request is valid.
	if err := request.ensureValid(); err != nil {
		return fmt.Errorf("invalid stage request: %w", err)
	}

	// Begin staging.
	paths, signatures, receiver, err := s.endpoint.Stage(request.Paths, request.Digests)
	if err != nil {
		s.encoder.Encode(&StageResponse{Error: err.Error()})
		return fmt.Errorf("unable to begin staging: %w", err)
	}

	// Send the response.
	response := &StageResponse{
		Paths:      paths,
		Signatures: signatures,
	}
	if err = s.encoder.Encode(response); err != nil {
		return fmt.Errorf("unable to send stage response: %w", err)
	}

	// If there weren't any paths requiring staging, then we're done.
	if len(paths) == 0 {
		return nil
	}

	// The remote side of the connection should now forward rsync operations, so
	// we need to decode and forward them to the receiver. If this operation
	// completes successfully, staging is complete and successful.
	decoder := newProtobufRsyncDecoder(s.decoder)
	if err = rsync.DecodeToReceiver(decoder, uint64(len(paths)), receiver); err != nil {
		return fmt.Errorf("unable to decode and forward rsync operations: %w", err)
	}

	// Success.
	return nil
}

// serveSupply serves a supply request.
func (s *endpointServer) serveSupply(request *SupplyRequest) error {
	// Ensure the request is valid.
	if err := request.ensureValid(); err != nil {
		return fmt.Errorf("invalid supply request: %w", err)
	}

	// Create an encoding receiver that can transmit rsync operations to the
	// remote.
	encoder := newProtobufRsyncEncoder(s.encoder)
	receiver := rsync.NewEncodingReceiver(encoder)

	// Perform supplying.
	if err := s.endpoint.Supply(request.Paths, request.Signatures, receiver); err != nil {
		return fmt.Errorf("unable to perform supplying: %w", err)
	}

	// Success.
	return nil
}

// serveTransitino serves a transition request.
func (s *endpointServer) serveTransition(request *TransitionRequest) error {
	// Ensure the request is valid.
	if err := request.ensureValid(); err != nil {
		return fmt.Errorf("invalid transition request: %w", err)
	}

	// Create a cancellable context for executing the transition.
	ctx, cancel := context.WithCancel(context.Background())

	// Start a Goroutine to watch for the completion request.
	completionReceiveErrors := make(chan error, 1)
	go func() {
		request := &TransitionCompletionRequest{}
		if err := s.decoder.Decode(request); err != nil {
			completionReceiveErrors <- fmt.Errorf("unable to receive completion request: %w", err)
		} else if err = request.ensureValid(); err != nil {
			completionReceiveErrors <- fmt.Errorf("received invalid completion request: %w", err)
		} else {
			completionReceiveErrors <- nil
		}
	}()

	// Start a Goroutine to execute the transition and send a response when
	// done.
	responseSendErrors := make(chan error, 1)
	go func() {
		// Perform the transition and set up the response.
		var response *TransitionResponse
		results, problems, stagerMissingFiles, err := s.endpoint.Transition(ctx, request.Transitions)
		if err != nil {
			response = &TransitionResponse{
				Error: err.Error(),
			}
		} else {
			// HACK: Wrap the results in Archives since Protocol Buffers can't
			// encode nil pointers in the result array.
			wrappedResults := make([]*core.Archive, len(results))
			for r, result := range results {
				wrappedResults[r] = &core.Archive{Content: result}
			}
			response = &TransitionResponse{
				Results:            wrappedResults,
				Problems:           problems,
				StagerMissingFiles: stagerMissingFiles,
			}
		}

		// Send the response.
		if err := s.encoder.Encode(response); err != nil {
			responseSendErrors <- fmt.Errorf("unable to transmit response: %w", err)
		} else {
			responseSendErrors <- nil
		}
	}()

	// Wait for both a completion request to be received and a response to be
	// sent. Both of these will occur, though their order is not known. If the
	// completion request is received first, then we cancel the subcontext to
	// preempt the transition and force transmission of a response. If the
	// response is sent first, then we know the completion request is on its
	// way. In this case, we still cancel the subcontext we created as required
	// by the context package to avoid leaking resources.
	var responseSendErr, completionReceiveErr error
	select {
	case completionReceiveErr = <-completionReceiveErrors:
		cancel()
		responseSendErr = <-responseSendErrors
	case responseSendErr = <-responseSendErrors:
		cancel()
		completionReceiveErr = <-completionReceiveErrors
	}

	// Check for errors.
	if responseSendErr != nil {
		return responseSendErr
	} else if completionReceiveErr != nil {
		return completionReceiveErr
	}

	// Success.
	return nil
}
