package session

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	syncpkg "sync"
	"time"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/encoding"
	"github.com/havoc-io/mutagen/filesystem"
	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/rsync"
	streampkg "github.com/havoc-io/mutagen/stream"
	"github.com/havoc-io/mutagen/sync"
)

const (
	endpointMethodInitialize = "endpoint.Initialize"
	endpointMethodScan       = "endpoint.Scan"
	endpointMethodTransmit   = "endpoint.Transmit"
	endpointMethodStage      = "endpoint.Stage"
	endpointMethodTransition = "endpoint.Transition"

	maxOutstandingStagingRequests = 4
)

func ServeEndpoint(stream io.ReadWriteCloser) error {
	// Perform housekeeping.
	housekeep()

	// Create a multiplexer. Ensure that it's closed when we're done serving.
	multiplexer := streampkg.Multiplex(stream, true)
	defer multiplexer.Close()

	// Create an RPC client to connect to the other endpoint.
	client := rpc.NewClient(multiplexer)

	// Create an RPC server.
	server := rpc.NewServer()

	// Create and register the endpoint.
	endpoint := newEndpoint(client)
	server.Register(endpoint)

	// Serve RPC requests until there is an error accepting new streams.
	return errors.Wrap(server.Serve(multiplexer), "error serving RPC requests")
}

type endpoint struct {
	client *rpc.Client
	syncpkg.RWMutex
	session   string
	version   Version
	root      string
	ignores   []string
	alpha     bool
	cachePath string
	cache     *sync.Cache
}

func newEndpoint(client *rpc.Client) *endpoint {
	return &endpoint{
		client: client,
	}
}

func (e *endpoint) Methods() map[string]rpc.Handler {
	return map[string]rpc.Handler{
		endpointMethodInitialize: e.initialize,
		endpointMethodScan:       e.scan,
		endpointMethodTransmit:   e.transmit,
		endpointMethodStage:      e.stage,
		endpointMethodTransition: e.transition,
	}
}

func (e *endpoint) initialize(stream rpc.HandlerStream) error {
	// Receive the request.
	var request initializeRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Lock the endpoint and defer its release.
	e.Lock()
	defer e.Unlock()

	// If we're already initialized, we can't do it again.
	if e.version != Version_Unknown {
		return errors.New("endpoint already initialized")
	}

	// Validate the request.
	if request.Session == "" {
		return errors.New("empty session identifier")
	} else if !request.Version.supported() {
		return errors.New("unsupported session version")
	} else if request.Root == "" {
		return errors.New("empty root path")
	}

	// Expand and normalize the root path.
	root, err := filesystem.Normalize(request.Root)
	if err != nil {
		return errors.Wrap(err, "unable to normalize root path")
	}

	// Compute the cache path.
	cachePath, err := pathForCache(request.Session, request.Alpha)
	if err != nil {
		return errors.Wrap(err, "unable to compute/create cache path")
	}

	// Load any existing cache. If it fails, just replace it with an empty one.
	cache := &sync.Cache{}
	if encoding.LoadAndUnmarshalProtobuf(cachePath, cache) != nil {
		cache = &sync.Cache{}
	}

	// Record initialization.
	e.session = request.Session
	e.version = request.Version
	e.root = root
	e.ignores = request.Ignores
	e.alpha = request.Alpha
	e.cachePath = cachePath
	e.cache = cache

	// Send the initialization response.
	return stream.Send(initializeResponse{
		PreservesExecutability: filesystem.PreservesExecutability,
	})
}

func (e *endpoint) scan(stream rpc.HandlerStream) error {
	// Receive the request.
	var request scanRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Lock the endpoint and defer its release.
	e.Lock()
	defer e.Unlock()

	// If we're not initialized, we can't do anything.
	if e.version == Version_Unknown {
		return errors.New("endpoint not initialized")
	}

	// Create a hasher.
	hasher := e.version.hasher()

	// Create a ticker to trigger polling at regular intervals. Ensure that it's
	// cancelled when we're done.
	ticker := time.NewTicker(scanPollInterval)
	defer ticker.Stop()

	// Set up a cancellable watch and ensure that it's cancelled when this
	// handler exits. We don't monitor for watch failure, because it might fail
	// in perfectly reasonable circumstances (e.g. the path not existing). In
	// that case we have to fall back to polling.
	watchContext, watchCancel := context.WithCancel(context.Background())
	watchEvents := make(chan struct{}, watchEventsBufferSize)
	go watch(watchContext, e.root, watchEvents)
	defer watchCancel()

	// Create a Goroutine that'll monitor for force requests. It will die once
	// the stream is closed, which will happen automatically once the handler
	// returns. If it fails before receiving a force request, it closes the
	// forces channel (in which case the loop should abort if it's still running
	// because something is wrong with the stream), otherwise it sends an empty
	// value (in which case the loop should force the response).
	forces := make(chan struct{}, 1)
	go func() {
		var forceRequest scanRequest
		if stream.Receive(&forceRequest) != nil {
			close(forces)
		} else {
			forces <- struct{}{}
		}
	}()

	// Loop until we're done.
	forced := false
	for {
		// Create a snapshot.
		// HACK: Concurrent modifications can cause scans to fail. That's simply
		// a fact of life and there's nothing we can do within the sync.Scan
		// function to deal with this. So if we receive an error from scan,
		// don't error out of the handler (which would cause the controller to
		// assume something very bad had happened and reconnect), but instead
		// tell the controller that everything is okay and that it should simply
		// wait and try to scan again.
		snapshot, cache, err := sync.Scan(e.root, hasher, e.cache, e.ignores)
		if err != nil {
			return stream.Send(scanResponse{TryAgain: true})
		}

		// Store the cache.
		if err := encoding.MarshalAndSaveProtobuf(e.cachePath, cache); err != nil {
			return errors.Wrap(err, "unable to save cache")
		}
		e.cache = cache

		// Marshal the snapshot.
		snapshotBytes, err := stableMarshal(snapshot)
		if err != nil {
			return errors.Wrap(err, "unable to marshal snapshot")
		}

		// Compute its checksum.
		snapshotChecksum := checksum(snapshotBytes)

		// If we've been forced or the checksum differs, send the snapshot.
		if forced || !bytes.Equal(snapshotChecksum, request.ExpectedSnapshotChecksum) {
			return stream.Send(scanResponse{
				SnapshotChecksum: snapshotChecksum,
				SnapshotDelta: rsync.DeltafyBytes(
					snapshotBytes,
					request.BaseSnapshotSignature,
				),
			})
		}

		// Otherwise, wait until an event occurs that makes us re-scan.
		select {
		case <-ticker.C:
		case <-watchEvents:
		case _, ok := <-forces:
			if !ok {
				return errors.New("error waiting for force request")
			}
			forced = true
		}
	}
}

func (e *endpoint) transmit(stream rpc.HandlerStream) error {
	// Receive the request.
	var request transmitRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Lock the endpoint for reading (to allow for concurrent transmissions) and
	// defer its release.
	e.RLock()
	defer e.RUnlock()

	// If we're not initialized, we can't do anything.
	if e.version == Version_Unknown {
		return errors.New("endpoint not initialized")
	}

	// Open the file and ensure it's closed when we're done.
	file, err := os.Open(filepath.Join(e.root, request.Path))
	if err != nil {
		return errors.Wrap(err, "unable to open source file")
	}
	defer file.Close()

	// Create an operation transmitter.
	transmit := func(operation rsync.Operation) error {
		return stream.Send(transmitResponse{Operation: operation})
	}

	// Transmit the delta.
	if err := rsync.Deltafy(file, request.BaseSignature, transmit); err != nil {
		return errors.Wrap(err, "unable to transmit delta")
	}

	// Success. We signal the end of the stream by closing it (which sends an
	// io.EOF), and returning from the handler will do that by default.
	return nil
}

type readSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

type emptyReadSeekCloser struct {
	*bytes.Reader
}

func newEmptyReadSeekCloser() readSeekCloser {
	return &emptyReadSeekCloser{bytes.NewReader(nil)}
}

func (e *emptyReadSeekCloser) Close() error {
	return nil
}

type stagingOperation struct {
	sync.StagingOperation
	base         readSeekCloser
	transmission rpc.ClientStream
}

func (e *endpoint) dispatch(
	context context.Context,
	queued <-chan stagingOperation,
	dispatched chan<- stagingOperation,
) error {
	// Loop over queued operations.
	for operation := range queued {
		// Attempt to open the base. If this fails (which it might if the file
		// doesn't exist), then simply use an empty base.
		if file, err := os.Open(filepath.Join(e.root, operation.Path)); err != nil {
			operation.base = newEmptyReadSeekCloser()
		} else {
			operation.base = file
		}

		// Compute the base signature. If there is an error, just abort, because
		// most likely the file is being modified concurrently and we'll have to
		// stage again later. We don't treat this as terminal though.
		baseSignature, err := rsync.Signature(operation.base)
		if err != nil {
			operation.base.Close()
			continue
		}

		// Invoke transmission. If this fails, something is probably wrong with
		// the network, so abort completely.
		transmission, err := e.client.Invoke(endpointMethodTransmit)
		if err != nil {
			operation.base.Close()
			return errors.Wrap(err, "unable to invoke transmission")
		}
		operation.transmission = transmission

		// Send the transmit request.
		request := transmitRequest{
			Path:          operation.Path,
			BaseSignature: baseSignature,
		}
		if err := transmission.Send(request); err != nil {
			operation.base.Close()
			operation.transmission.Close()
			return errors.Wrap(err, "unable to send transmission request")
		}

		// Add the operation to the dispatched queue while watching for
		// cancellation.
		select {
		case dispatched <- operation:
		case <-context.Done():
			operation.base.Close()
			operation.transmission.Close()
			return errors.New("dispatch cancelled")
		}
	}

	// Close the dispatched channel to indicate completion to the receiver.
	close(dispatched)

	// Success.
	return nil
}

func (e *endpoint) receive(
	context context.Context,
	dispatched <-chan stagingOperation,
	updater func(StagingStatus) error,
	total uint64,
) error {
	// Compute the staging root. We'll use this as our temporary directory.
	stagingRoot, err := pathForStagingRoot(e.session, e.alpha)
	if err != nil {
		return errors.Wrap(err, "unable to compute staging root")
	}

	// Track the progress index.
	index := uint64(0)

	// Loop until we're out of operations or cancelled.
	for {
		// Grab the next operation.
		var path string
		var entry *sync.Entry
		var base readSeekCloser
		var transmission rpc.ClientStream
		select {
		case operation, ok := <-dispatched:
			if ok {
				path = operation.Path
				entry = operation.Entry
				base = operation.base
				transmission = operation.transmission
			} else {
				return nil
			}
		case <-context.Done():
			return errors.New("receive cancelled")
		}

		// Increment our progress and send an update.
		index += 1
		if err := updater(StagingStatus{path, index, total}); err != nil {
			base.Close()
			transmission.Close()
			return errors.Wrap(err, "unable to send staging update")
		}

		// Create a temporary file in the staging directory and compute its
		// name.
		temporary, err := ioutil.TempFile(stagingRoot, "staging")
		if err != nil {
			base.Close()
			transmission.Close()
			return errors.Wrap(err, "unable to create temporary file")
		}
		temporaryPath := temporary.Name()

		// Create an rsync operation receiver. Note that this needs to pass back
		// an io.EOF that it receives directly in order to inform rsync.Patch
		// that operations are complete, so we don't want to wrap this error.
		receive := func() (rsync.Operation, error) {
			var response transmitResponse
			if err := transmission.Receive(&response); err != nil {
				return rsync.Operation{}, err
			}
			return response.Operation, nil
		}

		// Create a verification hasher.
		hasher := e.version.hasher()

		// Apply patch operations.
		err = rsync.Patch(temporary, base, receive, hasher)

		// Close files.
		temporary.Close()
		base.Close()

		// Close the transmission stream.
		transmission.Close()

		// If there was a patching error, remove the file. We don't abort the
		// staging pipeline in this case, because it's possible that the base
		// was being concurrently modified or that the remote file had some
		// error (perhaps also due to concurrent modification) and that future
		// receives won't fail.
		if err != nil {
			os.Remove(temporaryPath)
			continue
		}

		// Verify that the file contents match the expected digest. We don't
		// abort the pipeline on mismatch because it could be due to concurrent
		// modification.
		if !bytes.Equal(hasher.Sum(nil), entry.Digest) {
			os.Remove(temporaryPath)
			continue
		}

		// Set the file permissions.
		permissions := os.FileMode(0600)
		if entry.Executable {
			permissions = os.FileMode(0700)
		}
		if err = os.Chmod(temporaryPath, permissions); err != nil {
			os.Remove(temporaryPath)
			return errors.Wrap(err, "unable to set file permissions")
		}

		// Compute the staging path for the file.
		stagingPath, err := pathForStaging(e.session, e.alpha, path, entry)
		if err != nil {
			os.Remove(temporaryPath)
			return errors.Wrap(err, "unable to compute staging destination")
		}

		// Move the file into place.
		if err = os.Rename(temporaryPath, stagingPath); err != nil {
			os.Remove(temporaryPath)
			return errors.Wrap(err, "unable to relocate staging file")
		}
	}
}

func (e *endpoint) stage(stream rpc.HandlerStream) error {
	// Receive the request.
	var request stageRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Lock the endpoint reading (because we want to allow for concurrent
	// transmission operations) and defer its release.
	e.RLock()
	defer e.RUnlock()

	// If we're not initialized, we can't do anything.
	if e.version == Version_Unknown {
		return errors.New("endpoint not initialized")
	}

	// Compute the staging operations that we'll need to perform.
	operations, err := sync.StagingOperationsForChanges(request.Transitions)
	if err != nil {
		return errors.Wrap(err, "unable to compute staging operations")
	}

	// Create private wrappers for these operations and queue them up.
	queued := make(chan stagingOperation, len(operations))
	for _, o := range operations {
		queued <- stagingOperation{o, nil, nil}
	}
	close(queued)

	// Create a cancellable context in which our dispatch/receive operations
	// will execute.
	context, cancel := context.WithCancel(context.Background())

	// Create a queue of dispatched operations awaiting response.
	dispatched := make(chan stagingOperation, maxOutstandingStagingRequests)

	// Create a function to transmit staging status updates.
	updater := func(s StagingStatus) error {
		return stream.Send(stageResponse{Status: s})
	}

	// Start our dispatching/receiving pipeline.
	dispatchErrors := make(chan error, 1)
	receiveErrors := make(chan error, 1)
	go func() {
		dispatchErrors <- e.dispatch(context, queued, dispatched)
	}()
	go func() {
		receiveErrors <- e.receive(context, dispatched, updater, uint64(len(operations)))
	}()

	// Wait for completion from all of the Goroutines. If an error is received,
	// cancel the pipeline and wait for completion. Only record the first error,
	// because we don't want cancellation errors being returned.
	var dispatchDone, receiveDone bool
	var pipelineError error
	for !dispatchDone || !receiveDone {
		select {
		case err := <-dispatchErrors:
			dispatchDone = true
			if err != nil {
				if pipelineError == nil {
					pipelineError = errors.Wrap(err, "dispatch error")
				}
				cancel()
			}
		case err := <-receiveErrors:
			receiveDone = true
			if err != nil {
				if pipelineError == nil {
					pipelineError = errors.Wrap(err, "receive error")
				}
				cancel()
			}
		}
	}

	// Handle any pipeline error. If there was a pipeline error, then there may
	// be outstanding operations in the dispatched queue with open files, so we
	// need to close those out.
	if pipelineError != nil {
		for o := range dispatched {
			o.base.Close()
			o.transmission.Close()
		}
		return pipelineError
	}

	// Success. We signal the end of the stream by closing it (which sends an
	// io.EOF), and returning from the handler will do that by default.
	return nil
}

func (e *endpoint) transition(stream rpc.HandlerStream) error {
	// Receive the request.
	var request transitionRequest
	if err := stream.Receive(&request); err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Lock the endpoint and defer its release.
	e.Lock()
	defer e.Unlock()

	// If we're not initialized, we can't do anything.
	if e.version == Version_Unknown {
		return errors.New("endpoint not initialized")
	}

	// Create a staging provider.
	provider := func(path string, entry *sync.Entry) (string, error) {
		return pathForStaging(e.session, e.alpha, path, entry)
	}

	// Perform transitions.
	changes, problems := sync.Transition(e.root, request.Transitions, e.cache, provider)

	// Wipe the staging directory. Ignore any errors that occur, because we need
	// to return the transition results. If errors are occuring, they'll be
	// detected during the next round of staging.
	if stagingRoot, err := pathForStagingRoot(e.session, e.alpha); err == nil {
		os.RemoveAll(stagingRoot)
	}

	// Done.
	return stream.Send(transitionResponse{
		Changes:  changes,
		Problems: problems,
	})
}
