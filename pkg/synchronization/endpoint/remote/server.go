package remote

import (
	"context"
	"net"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/proto"

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
func ServeEndpoint(logger *logging.Logger, connection net.Conn, options ...EndpointServerOption) error {
	// Defer closure of the connection.
	defer connection.Close()

	// Enable read/write compression on the connection.
	reader := compression.NewDecompressingReader(connection)
	writer := compression.NewCompressingWriter(connection)

	// Create an encoder and decoder.
	encoder := encoding.NewProtobufEncoder(writer)
	decoder := encoding.NewProtobufDecoder(reader)

	// Create an endpoint configuration and apply all options.
	endpointServerOptions := &endpointServerOptions{}
	for _, o := range options {
		o.apply(endpointServerOptions)
	}

	// Receive the initialize request. If this fails, then send a failure
	// response (even though the pipe is probably broken) and abort.
	request := &InitializeSynchronizationRequest{}
	if err := decoder.Decode(request); err != nil {
		err = errors.Wrap(err, "unable to receive initialize request")
		encoder.Encode(&InitializeSynchronizationResponse{Error: err.Error()})
		return err
	}

	// If a root path override has been specified, then apply it.
	if endpointServerOptions.root != "" {
		request.Root = endpointServerOptions.root
	}

	// If configuration overrides have been provided, then validate them and
	// merge them into the main configuration.
	if endpointServerOptions.configuration != nil {
		if err := endpointServerOptions.configuration.EnsureValid(true); err != nil {
			err = errors.Wrap(err, "override configuration invalid")
			encoder.Encode(&InitializeSynchronizationResponse{Error: err.Error()})
			return err
		}
		request.Configuration = synchronization.MergeConfigurations(
			request.Configuration,
			endpointServerOptions.configuration,
		)
	}

	// If a connection validator has been provided, then ensure that it
	// approves if the specified endpoint configuration.
	if endpointServerOptions.connectionValidator != nil {
		err := endpointServerOptions.connectionValidator(
			request.Root,
			request.Session,
			request.Version,
			request.Configuration,
			request.Alpha,
		)
		if err != nil {
			err = errors.Wrap(err, "endpoint configuration rejected")
			encoder.Encode(&InitializeSynchronizationResponse{Error: err.Error()})
			return err
		}
	}

	// Ensure that the initialization request is valid.
	if err := request.ensureValid(); err != nil {
		err = errors.Wrap(err, "invalid initialize request")
		encoder.Encode(&InitializeSynchronizationResponse{Error: err.Error()})
		return err
	}

	// Expand and normalize the root path.
	if r, err := filesystem.Normalize(request.Root); err != nil {
		err = errors.Wrap(err, "unable to normalize synchronization root")
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
		endpointServerOptions.endpointOptions...,
	)
	if err != nil {
		err = errors.Wrap(err, "unable to create underlying endpoint")
		encoder.Encode(&InitializeSynchronizationResponse{Error: err.Error()})
		return err
	}
	defer endpoint.Shutdown()

	// Send a successful initialize response.
	if err = encoder.Encode(&InitializeSynchronizationResponse{}); err != nil {
		return errors.Wrap(err, "unable to send initialize response")
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
			return errors.Wrap(err, "unable to receive request")
		} else if err = request.ensureValid(); err != nil {
			return errors.Wrap(err, "invalid endpoint request")
		}

		// Handle the request based on type.
		if request.Poll != nil {
			if err := s.servePoll(request.Poll); err != nil {
				return errors.Wrap(err, "unable to serve poll request")
			}
		} else if request.Scan != nil {
			if err := s.serveScan(request.Scan); err != nil {
				return errors.Wrap(err, "unable to serve scan request")
			}
		} else if request.Stage != nil {
			if err := s.serveStage(request.Stage); err != nil {
				return errors.Wrap(err, "unable to serve stage request")
			}
		} else if request.Supply != nil {
			if err := s.serveSupply(request.Supply); err != nil {
				return errors.Wrap(err, "unable to serve supply request")
			}
		} else if request.Transition != nil {
			if err := s.serveTransition(request.Transition); err != nil {
				return errors.Wrap(err, "unable to serve transition request")
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
		return errors.Wrap(err, "invalid poll request")
	}

	// Create a cancellable context for executing the poll.
	ctx, cancel := context.WithCancel(context.Background())

	// Start a Goroutine to watch for the completion request.
	completionReceiveErrors := make(chan error, 1)
	go func() {
		request := &PollCompletionRequest{}
		if err := s.decoder.Decode(request); err != nil {
			completionReceiveErrors <- errors.Wrap(err, "unable to receive completion request")
		} else if err = request.ensureValid(); err != nil {
			completionReceiveErrors <- errors.Wrap(err, "received invalid completion request")
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
			responseSendErrors <- errors.Wrap(err, "unable to transmit response")
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
		return errors.Wrap(err, "invalid scan request")
	}

	// Create a cancellable context for executing the scan.
	ctx, cancel := context.WithCancel(context.Background())

	// Start a Goroutine to watch for the completion request.
	completionReceiveErrors := make(chan error, 1)
	go func() {
		request := &ScanCompletionRequest{}
		if err := s.decoder.Decode(request); err != nil {
			completionReceiveErrors <- errors.Wrap(err, "unable to receive completion request")
		} else if err = request.ensureValid(); err != nil {
			completionReceiveErrors <- errors.Wrap(err, "received invalid completion request")
		} else {
			completionReceiveErrors <- nil
		}
	}()

	// Start a Goroutine to execute the scan and send a response when done.
	responseSendErrors := make(chan error, 1)
	go func() {
		// Create a deterministic Protocol Buffers marshaller.
		buffer := proto.NewBuffer(nil)
		buffer.SetDeterministic(true)

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
		} else if err = buffer.Marshal(&core.Archive{Root: snapshot}); err != nil {
			response = &ScanResponse{
				Error: errors.Wrap(err, "unable to marshal snapshot").Error(),
			}
		} else {
			snapshotBytes := buffer.Bytes()
			delta := engine.DeltafyBytes(snapshotBytes, request.BaseSnapshotSignature, 0)
			response = &ScanResponse{
				SnapshotDelta:          delta,
				PreservesExecutability: preservesExecutability,
			}
		}

		// Send the response.
		if err := s.encoder.Encode(response); err != nil {
			responseSendErrors <- errors.Wrap(err, "unable to transmit response")
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
		return errors.Wrap(err, "invalid stage request")
	}

	// Begin staging.
	paths, signatures, receiver, err := s.endpoint.Stage(request.Paths, request.Digests)
	if err != nil {
		s.encoder.Encode(&StageResponse{Error: err.Error()})
		return errors.Wrap(err, "unable to begin staging")
	}

	// Send the response.
	response := &StageResponse{
		Paths:      paths,
		Signatures: signatures,
	}
	if err = s.encoder.Encode(response); err != nil {
		return errors.Wrap(err, "unable to send stage response")
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
		return errors.Wrap(err, "unable to decode and forward rsync operations")
	}

	// Success.
	return nil
}

// serveSupply serves a supply request.
func (s *endpointServer) serveSupply(request *SupplyRequest) error {
	// Ensure the request is valid.
	if err := request.ensureValid(); err != nil {
		return errors.Wrap(err, "invalid supply request")
	}

	// Create an encoding receiver that can transmit rsync operations to the
	// remote.
	encoder := newProtobufRsyncEncoder(s.encoder)
	receiver := rsync.NewEncodingReceiver(encoder)

	// Perform supplying.
	if err := s.endpoint.Supply(request.Paths, request.Signatures, receiver); err != nil {
		return errors.Wrap(err, "unable to perform supplying")
	}

	// Success.
	return nil
}

// serveTransitino serves a transition request.
func (s *endpointServer) serveTransition(request *TransitionRequest) error {
	// Ensure the request is valid.
	if err := request.ensureValid(); err != nil {
		return errors.Wrap(err, "invalid transition request")
	}

	// Create a cancellable context for executing the transition.
	ctx, cancel := context.WithCancel(context.Background())

	// Start a Goroutine to watch for the completion request.
	completionReceiveErrors := make(chan error, 1)
	go func() {
		request := &TransitionCompletionRequest{}
		if err := s.decoder.Decode(request); err != nil {
			completionReceiveErrors <- errors.Wrap(err, "unable to receive completion request")
		} else if err = request.ensureValid(); err != nil {
			completionReceiveErrors <- errors.Wrap(err, "received invalid completion request")
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
				wrappedResults[r] = &core.Archive{Root: result}
			}
			response = &TransitionResponse{
				Results:            wrappedResults,
				Problems:           problems,
				StagerMissingFiles: stagerMissingFiles,
			}
		}

		// Send the response.
		if err := s.encoder.Encode(response); err != nil {
			responseSendErrors <- errors.Wrap(err, "unable to transmit response")
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
