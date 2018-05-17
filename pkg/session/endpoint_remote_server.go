package session

import (
	contextpkg "context"
	"encoding/gob"
	"net"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/proto"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/rsync"
)

type remoteEndpointServer struct {
	// encoder is the control stream encoder.
	encoder *gob.Encoder
	// decoder is the control stream decoder.
	decoder *gob.Decoder
	// endpoint is the underlying local endpoint.
	endpoint endpoint
}

// TODO: Document that the provided streams should be closed (in a manner that
// unblocks them) when the function returns in order to ensure that all
// Goroutines have exited (they could be blocked in encodes/decodes - we only
// exit after the first fails). We could try to pass in a io.Closer here, but in
// the agent it would have to be closing standard input/output/error, and OS
// pipes can (depending on the platform) block on close if you try to close
// while in a read or write, so it's better that the caller just ensures the
// streams are closed, in this case by exiting the process.
func ServeEndpoint(connection net.Conn) error {
	// Defer closure of the connection.
	defer connection.Close()

	// Create encoders and decoders.
	encoder := gob.NewEncoder(connection)
	decoder := gob.NewDecoder(connection)

	// Receive the initialize request. If this fails, then send a failure
	// response (even though the pipe is probably broken) and abort.
	var request initializeRequest
	if err := decoder.Decode(&request); err != nil {
		err = errors.Wrap(err, "unable to receive initialize request")
		encoder.Encode(initializeResponse{Error: err.Error()})
		return err
	}

	// Create the underlying endpoint. If it fails to create, then send a
	// failure response and abort. If it succeeds, then defer its closure.
	endpoint, err := newLocalEndpoint(request.Session, request.Version, request.Root, request.Ignores, request.Alpha)
	if err != nil {
		err = errors.Wrap(err, "unable to create underlying endpoint")
		encoder.Encode(initializeResponse{Error: err.Error()})
		return err
	}
	defer endpoint.shutdown()

	// Send a successful initialize response.
	response := initializeResponse{PreservesExecutability: filesystem.PreservesExecutability}
	if err = encoder.Encode(response); err != nil {
		return errors.Wrap(err, "unable to send initialize response")
	}

	// Create the server.
	server := &remoteEndpointServer{
		endpoint: endpoint,
		encoder:  encoder,
		decoder:  decoder,
	}

	// Server until an error occurs.
	return server.serve()
}

func (s *remoteEndpointServer) serve() error {
	// Receive and process control requests until there's an error.
	for {
		// Receive the next request.
		var request endpointRequest
		if err := s.decoder.Decode(&request); err != nil {
			return errors.Wrap(err, "unable to receive request")
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
			return errors.New("invalid request")
		}
	}
}

func (s *remoteEndpointServer) servePoll(_ *pollRequest) error {
	// Create a cancellable context for executing the poll. The context may be
	// cancelled to force a response, but in case the response comes naturally,
	// ensure the context is cancelled before we're done to avoid leaking a
	// Goroutine.
	pollContext, forceResponse := contextpkg.WithCancel(contextpkg.Background())
	defer forceResponse()

	// Start a Goroutine to execute the poll and send a response when done.
	responseSendResults := make(chan error, 1)
	go func() {
		if err := s.endpoint.poll(pollContext); err != nil {
			s.encoder.Encode(pollResponse{Error: err.Error()})
			responseSendResults <- errors.Wrap(err, "polling error")
		}
		responseSendResults <- errors.Wrap(
			s.encoder.Encode(pollResponse{}),
			"unable to send poll response",
		)
	}()

	// Start a Goroutine to watch for the done request.
	completionReceiveResults := make(chan error, 1)
	go func() {
		var request pollCompletionRequest
		completionReceiveResults <- errors.Wrap(
			s.decoder.Decode(&request),
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

func (s *remoteEndpointServer) serveScan(request *scanRequest) error {
	// Perform a scan. Passing a nil ancestor is fine - it just stops
	// executability propagation, but that will happen in the remoteEndpointClient
	// instance. If a retry is requested or an error occurs, send a response.
	snapshot, tryAgain, err := s.endpoint.scan(nil)
	if tryAgain {
		if err := s.encoder.Encode(scanResponse{TryAgain: true}); err != nil {
			return errors.Wrap(err, "unable to send scan retry response")
		}
		return nil
	} else if err != nil {
		s.encoder.Encode(scanResponse{Error: err.Error()})
		return errors.Wrap(err, "unable to perform scan")
	}

	// Marshal the snapshot in a deterministic fashion.
	buffer := proto.NewBuffer(nil)
	buffer.SetDeterministic(true)
	if err := buffer.Marshal(&Archive{Root: snapshot}); err != nil {
		return errors.Wrap(err, "unable to marshal snapshot")
	}
	snapshotBytes := buffer.Bytes()

	// Create an rsync engine.
	engine := rsync.NewEngine()

	// Compute the snapshot's delta against the base.
	delta := engine.DeltafyBytes(snapshotBytes, request.BaseSnapshotSignature, 0)

	// Send the response.
	response := scanResponse{SnapshotDelta: delta}
	if err := s.encoder.Encode(response); err != nil {
		return errors.Wrap(err, "unable to send scan response")
	}

	// Success.
	return nil
}

func (s *remoteEndpointServer) serveStage(request *stageRequest) error {
	// Validate the request internals since they came over the wire.
	for _, e := range request.Entries {
		if err := e.EnsureValid(); err != nil {
			err = errors.Wrap(err, "received invalid entry")
			s.encoder.Encode(stageResponse{Error: err.Error()})
			return err
		}
	}

	// Begin staging.
	paths, signatures, receiver, err := s.endpoint.stage(request.Paths, request.Entries)
	if err != nil {
		s.encoder.Encode(stageResponse{Error: err.Error()})
		return errors.Wrap(err, "unable to begin staging")
	}

	// Send the response.
	if err = s.encoder.Encode(stageResponse{Paths: paths, Signatures: signatures}); err != nil {
		return errors.Wrap(err, "unable to send stage response")
	}

	// The remote side of the connection should now forward rsync operations, so
	// we need to decode and forward them to the receiver. If this operation
	// completes successfully, staging is complete and successful.
	if err = rsync.DecodeToReceiver(s.decoder, uint64(len(paths)), receiver); err != nil {
		return errors.Wrap(err, "unable to decode and forward rsync operations")
	}

	// Success.
	return nil
}

func (s *remoteEndpointServer) serveSupply(request *supplyRequest) error {
	// Create an encoding receiver that can transmit rsync operations to the
	// remote.
	receiver := rsync.NewEncodingReceiver(s.encoder)

	// Perform supplying.
	if err := s.endpoint.supply(request.Paths, request.Signatures, receiver); err != nil {
		return errors.Wrap(err, "unable to perform supplying")
	}

	// Success.
	return nil
}

func (s *remoteEndpointServer) serveTransition(request *transitionRequest) error {
	// Validate the request internals since they came over the wire.
	for _, t := range request.Transitions {
		if err := t.EnsureValid(); err != nil {
			err = errors.Wrap(err, "received invalid transition")
			s.encoder.Encode(transitionResponse{Error: err.Error()})
			return err
		}
	}

	// Perform the transition.
	changes, problems, err := s.endpoint.transition(request.Transitions)
	if err != nil {
		s.encoder.Encode(transitionResponse{Error: err.Error()})
		return errors.Wrap(err, "unable to perform transition")
	}

	// Send the response.
	response := transitionResponse{Changes: changes, Problems: problems}
	if err = s.encoder.Encode(response); err != nil {
		return errors.Wrap(err, "unable to send transition response")
	}

	// Success.
	return nil
}
