package remote

import (
	contextpkg "context"
	"net"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/proto"

	"github.com/havoc-io/mutagen/pkg/compression"
	"github.com/havoc-io/mutagen/pkg/encoding"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/session/endpoint/local"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// endpointServer wraps a local endpoint instances and dispatches requests to
// this endpoint from an endpoint client.
type endpointServer struct {
	// encoder is the control stream encoder.
	encoder *encoding.ProtobufEncoder
	// decoder is the control stream decoder.
	decoder *encoding.ProtobufDecoder
	// endpoint is the underlying local endpoint.
	endpoint session.Endpoint
}

// ServeEndpoint creates and serves a remote endpoint server on the specified
// connection.
func ServeEndpoint(connection net.Conn, options ...EndpointServerOption) error {
	// Defer closure of the connection.
	defer connection.Close()

	// Send our magic number to the client.
	if err := sendMagicNumber(connection, serverMagicNumber); err != nil {
		return &handshakeTransportError{errors.Wrap(err, "unable to send server magic number")}
	}

	// Receive the client's magic number. We treat a mismatch of the magic
	// number as a transport error as well, because it indicates that we're not
	// actually talking to a Mutagen client.
	if magicOk, err := receiveAndCompareMagicNumber(connection, clientMagicNumber); err != nil {
		return &handshakeTransportError{errors.Wrap(err, "unable to receive client magic number")}
	} else if !magicOk {
		return &handshakeTransportError{errors.New("client magic number incorrect")}
	}

	// Send our version to the client.
	if err := mutagen.SendVersion(connection); err != nil {
		return &handshakeTransportError{errors.Wrap(err, "unable to send server version")}
	}

	// Receive the client's version.
	clientMajor, clientMinor, clientPatch, err := mutagen.ReceiveVersion(connection)
	if err != nil {
		return &handshakeTransportError{errors.Wrap(err, "unable to receive client version")}
	}

	// Ensure that our Mutagen versions are compatible. For now, we enforce that
	// they're equal.
	// TODO: Once we lock-in an internal protocol that we're going to support
	// for some time, we can allow some version skew. In the server side in
	// particular, we'll probably just want to ensure that the client version is
	// greater than the Mutagen version at which the protocol was locked in, and
	// perhaps also ensure that the client version isn't blacklisted, e.g. due
	// to a bug.
	versionMatch := clientMajor == mutagen.VersionMajor &&
		clientMinor == mutagen.VersionMinor &&
		clientPatch == mutagen.VersionPatch
	if !versionMatch {
		return errors.New("version mismatch")
	}

	// TODO: Finish version negotiation.

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
	request := &InitializeRequest{}
	if err := decoder.Decode(request); err != nil {
		err = errors.Wrap(err, "unable to receive initialize request")
		encoder.Encode(&InitializeResponse{Error: err.Error()})
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
			encoder.Encode(&InitializeResponse{Error: err.Error()})
			return err
		}
		request.Configuration = session.MergeConfigurations(
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
			encoder.Encode(&InitializeResponse{Error: err.Error()})
			return err
		}
	}

	// Ensure that the initialization request is valid.
	if err := request.ensureValid(); err != nil {
		err = errors.Wrap(err, "invalid initialize request")
		encoder.Encode(&InitializeResponse{Error: err.Error()})
		return err
	}

	// Create the underlying endpoint. If it fails to create, then send a
	// failure response and abort. If it succeeds, then defer its closure.
	endpoint, err := local.NewEndpoint(
		request.Root,
		request.Session,
		request.Version,
		request.Configuration,
		request.Alpha,
		endpointServerOptions.endpointOptions...,
	)
	if err != nil {
		err = errors.Wrap(err, "unable to create underlying endpoint")
		encoder.Encode(&InitializeResponse{Error: err.Error()})
		return err
	}
	defer endpoint.Shutdown()

	// Send a successful initialize response.
	if err = encoder.Encode(&InitializeResponse{}); err != nil {
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

	// Create a cancellable context for executing the poll. The context may be
	// cancelled to force a response, but in case the response comes naturally,
	// ensure the context is cancelled before we're done to avoid leaking a
	// Goroutine.
	pollContext, forceResponse := contextpkg.WithCancel(contextpkg.Background())
	defer forceResponse()

	// Start a Goroutine to execute the poll and send a response when done.
	responseSendResults := make(chan error, 1)
	go func() {
		if err := s.endpoint.Poll(pollContext); err != nil {
			s.encoder.Encode(&PollResponse{Error: err.Error()})
			responseSendResults <- errors.Wrap(err, "polling error")
		}
		responseSendResults <- errors.Wrap(
			s.encoder.Encode(&PollResponse{}),
			"unable to send poll response",
		)
	}()

	// Start a Goroutine to watch for the done request.
	completionReceiveResults := make(chan error, 1)
	go func() {
		request := &PollCompletionRequest{}
		completionReceiveResults <- errors.Wrap(
			s.decoder.Decode(request),
			"unable to receive completion request",
		)
	}()

	// Wait for both a completion request to be received and a response to be
	// sent. Both of these will happen, though their order is not guaranteed. If
	// the response has been sent, then we know the completion request is on its
	// way, so just wait for it. If the completion receive comes first, then
	// force the response and wait for it to be sent.
	var responseSendErr, completionReceiveErr error
	select {
	case responseSendErr = <-responseSendResults:
		completionReceiveErr = <-completionReceiveResults
	case completionReceiveErr = <-completionReceiveResults:
		forceResponse()
		responseSendErr = <-responseSendResults
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

	// Perform a scan. Passing a nil ancestor is fine - it's not used for local
	// endpoints anyway. If a retry is requested or an error occurs, send a
	// response.
	snapshot, preservesExecutability, err, tryAgain := s.endpoint.Scan(nil, request.Full)
	if tryAgain {
		response := &ScanResponse{
			Error:    err.Error(),
			TryAgain: true,
		}
		if err := s.encoder.Encode(response); err != nil {
			return errors.Wrap(err, "unable to send scan retry response")
		}
		return nil
	} else if err != nil {
		s.encoder.Encode(&ScanResponse{Error: err.Error()})
		return errors.Wrap(err, "unable to perform scan")
	}

	// Marshal the snapshot in a deterministic fashion.
	buffer := proto.NewBuffer(nil)
	buffer.SetDeterministic(true)
	if err := buffer.Marshal(&sync.Archive{Root: snapshot}); err != nil {
		return errors.Wrap(err, "unable to marshal snapshot")
	}
	snapshotBytes := buffer.Bytes()

	// Create an rsync engine.
	engine := rsync.NewEngine()

	// Compute the snapshot's delta against the base.
	delta := engine.DeltafyBytes(snapshotBytes, request.BaseSnapshotSignature, 0)

	// Send the response.
	response := &ScanResponse{
		SnapshotDelta:          delta,
		PreservesExecutability: preservesExecutability,
	}
	if err := s.encoder.Encode(response); err != nil {
		return errors.Wrap(err, "unable to send scan response")
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

	// Perform the transition.
	results, problems, stagerMissingFiles, err := s.endpoint.Transition(request.Transitions)
	if err != nil {
		s.encoder.Encode(&TransitionResponse{Error: err.Error()})
		return errors.Wrap(err, "unable to perform transition")
	}

	// HACK: Wrap the results in Archives since neither Protocol Buffers can't
	// encode nil pointers in the result array.
	wrappedResults := make([]*sync.Archive, len(results))
	for r, result := range results {
		wrappedResults[r] = &sync.Archive{Root: result}
	}

	// Send the response.
	response := &TransitionResponse{
		Results:            wrappedResults,
		Problems:           problems,
		StagerMissingFiles: stagerMissingFiles,
	}
	if err = s.encoder.Encode(response); err != nil {
		return errors.Wrap(err, "unable to send transition response")
	}

	// Success.
	return nil
}
