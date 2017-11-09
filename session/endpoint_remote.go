package session

import (
	"encoding/gob"
	"io"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/sync"
)

type initializeRequest struct {
	Session string
	Version Version
	Root    string
	Ignores []string
	Alpha   bool
}

type initializeResponse struct {
	PreservesExecutability bool
	Error                  string
}

type watchEvent struct{}

type scanRequest struct {
	BaseSnapshotSignature rsync.Signature
}

type scanResponse struct {
	TryAgain      bool
	SnapshotDelta []rsync.Operation
	Error         string
}

type stageRequest struct {
	Paths   []string
	Entries []*sync.Entry
}

type stageResponse struct {
	Paths      []string
	Signatures []rsync.Signature
	Error      string
}

type supplyRequest struct {
	Paths      []string
	Signatures []rsync.Signature
}

type transitionRequest struct {
	Transitions []sync.Change
}

type transitionResponse struct {
	Changes  []sync.Change
	Problems []sync.Problem
	Error    string
}

type endpointRequest struct {
	Scan       *scanRequest
	Stage      *stageRequest
	Supply     *supplyRequest
	Transition *transitionRequest
}

// TODO: Document that the provided streams should be closed (in a manner that
// unblocks them) when the function returns in order to ensure that all
// Goroutines have exited (they could be blocked in encodes/decodes - we only
// exit after the first fails). We could try to pass in a io.Closer here, but in
// the agent it would have to be closing standard input/output/error, and OS
// pipes can (depending on the platform) block on close if you try to close
// while in a read or write, so it's better that the caller just ensures the
// streams are closed, in this case by exiting the process.
func ServeEndpoint(controlStream io.ReadWriter, watchStream io.Writer) error {
	// Create encoders and decoders.
	encoder := gob.NewEncoder(controlStream)
	decoder := gob.NewDecoder(controlStream)
	watchEncoder := gob.NewEncoder(watchStream)

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
	defer endpoint.close()

	// Send a successful initialize response.
	response := initializeResponse{PreservesExecutability: filesystem.PreservesExecutability}
	if err = encoder.Encode(response); err != nil {
		return errors.Wrap(err, "unable to send initialize response")
	}

	// Serve watch events for the endpoint. This will exit when the events
	// channel is closed, which will happen when the underlying endpoint is
	// closed at the termination of ServeEndpoint. The watch is guaranteed not
	// to stop unless cancelled - it doesn't have any total failure modes. The
	// encoder may fail, so we have to track termination.
	watchForwardingErrors := make(chan error, 1)
	go func() {
		events := endpoint.poller()
		for _ = range events {
			if err := watchEncoder.Encode(watchEvent{}); err != nil {
				watchForwardingErrors <- err
				return
			}
		}
		watchForwardingErrors <- errors.New("watch ended")
	}()

	// Serve requests on the control channel. Any number of things may fail
	// here, so we have to track termination.
	requestServingErrors := make(chan error, 1)
	go func() {
		requestServingErrors <- serveEndpointRequests(endpoint, encoder, decoder)
	}()

	// Wait for failure from either serving Goroutine.
	select {
	case err := <-watchForwardingErrors:
		return errors.Wrap(err, "watching forwarding failed")
	case err := <-requestServingErrors:
		return errors.Wrap(err, "serving failed")
	}
}

func serveEndpointRequests(endpoint *localEndpoint, encoder *gob.Encoder, decoder *gob.Decoder) error {
	// Create an rsync engine for use in snapshot transmission.
	rsyncEngine := rsync.NewEngine()

	// Receive and process control requests until there's an error.
	for {
		// Receive the next request.
		var request endpointRequest
		if err := decoder.Decode(&request); err != nil {
			return errors.Wrap(err, "unable to receive request")
		}

		// Handle the request based on type.
		if request.Scan != nil {
			// Perform a scan. Passing a nil ancestor is fine - it just stops
			// executability propagation, but that will happen in the
			// remoteEndpoint instance. If a retry is requested or an error
			// occurs, send a response.
			snapshot, tryAgain, err := endpoint.scan(nil)
			if tryAgain {
				if err := encoder.Encode(scanResponse{TryAgain: true}); err != nil {
					return errors.Wrap(err, "unable to send scan retry response")
				}
				continue
			} else if err != nil {
				encoder.Encode(scanResponse{Error: err.Error()})
				return errors.Wrap(err, "unable to perform scan")
			}

			// Marshal the snapshot.
			snapshotBytes, err := marshalEntry(snapshot)
			if err != nil {
				return errors.Wrap(err, "unable to marshal snapshot")
			}

			// Compute it's delta against the base.
			delta := rsyncEngine.DeltafyBytes(snapshotBytes, request.Scan.BaseSnapshotSignature, 0)

			// Send the response.
			if err := encoder.Encode(scanResponse{SnapshotDelta: delta}); err != nil {
				return errors.Wrap(err, "unable to send scan response")
			}
		} else if request.Stage != nil {
			// Begin staging.
			paths, signatures, receiver, err := endpoint.stage(request.Stage.Paths, request.Stage.Entries)
			if err != nil {
				encoder.Encode(stageResponse{Error: err.Error()})
				return errors.Wrap(err, "unable to begin staging")
			}

			// Send the response.
			if err = encoder.Encode(stageResponse{Paths: paths, Signatures: signatures}); err != nil {
				return errors.Wrap(err, "unable to send stage response")
			}

			// The remote side of the connection should now forward rsync
			// operations, so we need to decode and forward them to the
			// receiver. If this operation completes successfully, staging is
			// complete and successful.
			if err = rsync.DecodeToReceiver(decoder, uint64(len(paths)), receiver); err != nil {
				return errors.Wrap(err, "unable to decode and forward rsync operations")
			}
		} else if request.Supply != nil {
			// Create an encoding receiver that can transmit rsync operations to
			// the remote.
			receiver := rsync.NewEncodingReceiver(encoder)

			// Perform supplying.
			if err := endpoint.supply(request.Supply.Paths, request.Supply.Signatures, receiver); err != nil {
				return errors.Wrap(err, "unable to perform supplying")
			}
		} else if request.Transition != nil {
			// Perform the transition.
			changes, problems, err := endpoint.transition(request.Transition.Transitions)
			if err != nil {
				encoder.Encode(transitionResponse{Error: err.Error()})
				return errors.Wrap(err, "unable to perform transition")
			}

			// Send the response.
			if err = encoder.Encode(transitionResponse{Changes: changes, Problems: problems}); err != nil {
				return errors.Wrap(err, "unable to send transition response")
			}
		} else {
			return errors.New("invalid request")
		}
	}
}

// remoteEndpoint is an endpoint implementation that provides a proxy for
// another endpoint over a network. It is designed to be paired with
// ServeEndpoint.
type remoteEndpoint struct {
	// closer closes the underlying streams.
	closer io.Closer
	// encoder is the encoder for the control stream.
	encoder *gob.Encoder
	// decoder is the decoder for the control stream.
	decoder *gob.Decoder
	// watchEvents is the channel of watch events forwarded from the remote.
	watchEvents chan struct{}
	// rsyncEngine is the rsync engine used for snapshot transfers.
	rsyncEngine *rsync.Engine
	// preservesExecutability indicates whether or not the remote endpoint
	// preserves executability.
	preservesExecutability bool
}

// newRemoteEndpoint constructs a new remote Endpoint instance using the
// specified streams. The provided io.Closer must unblock reads and writes on
// the control and watch streams when closed.
func newRemoteEndpoint(
	controlStream io.ReadWriter,
	watchStream io.Reader,
	closer io.Closer,
	session string,
	version Version,
	root string,
	ignores []string,
	alpha bool,
) (*remoteEndpoint, error) {
	// Create encoders and decoders.
	encoder := gob.NewEncoder(controlStream)
	decoder := gob.NewDecoder(controlStream)
	watchDecoder := gob.NewDecoder(watchStream)

	// Create and send the initialize request.
	request := initializeRequest{
		Session: session,
		Version: version,
		Root:    root,
		Ignores: ignores,
		Alpha:   alpha,
	}
	if err := encoder.Encode(request); err != nil {
		closer.Close()
		return nil, errors.Wrap(err, "unable to send initialize request")
	}

	// Receive the response and check for remote errors.
	var response initializeResponse
	if err := decoder.Decode(&response); err != nil {
		closer.Close()
		return nil, errors.Wrap(err, "unable to receive transition response")
	} else if response.Error != "" {
		closer.Close()
		return nil, errors.Errorf("remote error: %s", response.Error)
	}

	// Create a watch events channel and a Goroutine to forward events to it.
	// Close the channel if there are any decoding errors.
	watchEvents := make(chan struct{}, 1)
	go func() {
		for {
			var event watchEvent
			if watchDecoder.Decode(&event) != nil {
				close(watchEvents)
				return
			}
			select {
			case watchEvents <- struct{}{}:
			default:
			}
		}
	}()

	// Success.
	return &remoteEndpoint{
		closer:                 closer,
		encoder:                encoder,
		decoder:                decoder,
		watchEvents:            watchEvents,
		rsyncEngine:            rsync.NewEngine(),
		preservesExecutability: response.PreservesExecutability,
	}, nil
}

func (e *remoteEndpoint) poller() chan struct{} {
	return e.watchEvents
}

func (e *remoteEndpoint) scan(ancestor *sync.Entry) (*sync.Entry, bool, error) {
	// Marshal the ancestor and compute its rsync signature. We'll use it as a
	// base for an rsync transfer of the serialized snapshot.
	ancestorBytes, err := marshalEntry(ancestor)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to marshal ancestor")
	}
	ancestorSignature := e.rsyncEngine.BytesSignature(ancestorBytes, 0)

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
	snapshotBytes, err := e.rsyncEngine.PatchBytes(ancestorBytes, ancestorSignature, response.SnapshotDelta)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to patch base snapshot")
	}

	// Unmarshal the snapshot.
	snapshot, err := unmarshalEntry(snapshotBytes)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to unmarshal snapshot")
	}

	// If the endpoint doesn't preserve executability, then propagate
	// executability from the ancestor.
	if !e.preservesExecutability {
		snapshot = sync.PropagateExecutability(ancestor, snapshot)
	}

	// Success.
	return snapshot, false, nil
}

func (e *remoteEndpoint) stage(paths []string, entries []*sync.Entry) ([]string, []rsync.Signature, rsync.Receiver, error) {
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

func (e *remoteEndpoint) supply(paths []string, signatures []rsync.Signature, receiver rsync.Receiver) error {
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

func (e *remoteEndpoint) transition(transitions []sync.Change) ([]sync.Change, []sync.Problem, error) {
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

	// Success.
	return response.Changes, response.Problems, nil
}

func (e *remoteEndpoint) close() error {
	// Close the underlying connection. This will cause all stream reads/writes
	// to unblock, which will also cause the watch event forwarding Goroutine to
	// close the associated events channel and exit.
	return e.closer.Close()
}
