package session

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
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
	"github.com/havoc-io/mutagen/sync"
)

const (
	endpointMethodInitialize = "endpoint.Initialize"
	endpointMethodScan       = "endpoint.Scan"
	endpointMethodTransmit   = "endpoint.Transmit"
	endpointMethodStage      = "endpoint.Stage"
	endpointMethodApply      = "endpoint.Apply"

	cachesDirectoryName  = "caches"
	stagingDirectoryName = "staging"
	alphaName            = "alpha"
	betaName             = "beta"

	maxOutstandingStagingRequests = 4
)

func ServeEndpoint(stream io.ReadWriteCloser) error {
	// Create a multiplexer. Ensure that it's closed when we're done serving.
	multiplexer := multiplex(stream, true)
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
	version     Version
	root        string
	ignores     []string
	cachePath   string
	cache       *sync.Cache
	stagingPath string
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
		endpointMethodApply:      e.apply,
	}
}

func (e *endpoint) initialize(stream *rpc.HandlerStream) {
	// Create an error transmitter.
	sendError := func(err error) {
		stream.Encode(initializeResponse{Error: err.Error()})
	}

	// Receive the request.
	var request initializeRequest
	if err := stream.Decode(&request); err != nil {
		sendError(errors.Wrap(err, "unable to receive request"))
		return
	}

	// Lock the endpoint and defer its release.
	e.Lock()
	defer e.Unlock()

	// If we're already initialized, we can't do it again.
	if e.version != Version_Unknown {
		sendError(errors.New("endpoint already initialized"))
		return
	}

	// Validate the request.
	if request.Session == "" {
		sendError(errors.New("empty session identifier"))
		return
	} else if !request.Version.supported() {
		sendError(errors.New("unsupported session version"))
		return
	} else if request.Root == "" {
		sendError(errors.New("empty root path"))
		return
	}

	// Expand and normalize the root path.
	root, err := filesystem.Normalize(request.Root)
	if err != nil {
		sendError(errors.Wrap(err, "unable to normalize root path"))
		return
	}

	// Compute the endpoint name.
	endpointName := alphaName
	if !request.Alpha {
		endpointName = betaName
	}

	// Compute the cache path.
	cachesDirectory, err := filesystem.Mutagen(cachesDirectoryName)
	if err != nil {
		sendError(errors.Wrap(err, "unable to compute/create caches path"))
		return
	}
	cacheName := fmt.Sprintf("%s_%s", request.Session, endpointName)
	cachePath := filepath.Join(cachesDirectory, cacheName)

	// Load any existing cache. If it fails, just replace it with an empty one.
	cache := &sync.Cache{}
	if encoding.LoadAndUnmarshalProtobuf(cachePath, cache) != nil {
		cache = &sync.Cache{}
	}

	// Compute and create the staging path.
	stagingPath, err := filesystem.Mutagen(
		stagingDirectoryName, request.Session, endpointName,
	)
	if err != nil {
		sendError(errors.Wrap(err, "unable to compute/create staging path"))
		return
	}

	// Record initialization.
	e.version = request.Version
	e.root = root
	e.ignores = request.Ignores
	e.cachePath = cachePath
	e.stagingPath = stagingPath

	// Send the initialization response.
	stream.Encode(initializeResponse{
		PreservesExecutability: filesystem.PreservesExecutability,
	})
}

func (e *endpoint) scan(stream *rpc.HandlerStream) {
	// Create an error transmitter.
	sendError := func(err error) {
		stream.Encode(scanResponse{Error: err.Error()})
	}

	// Receive the request.
	var request scanRequest
	if err := stream.Decode(&request); err != nil {
		sendError(errors.Wrap(err, "unable to decode request"))
		return
	}

	// Lock the endpoint and defer its release.
	e.Lock()
	defer e.Unlock()

	// If we're not initialized, we can't do anything.
	if e.version == Version_Unknown {
		sendError(errors.New("endpoint not initialized"))
		return
	}

	// Create a hasher.
	hasher, err := e.version.hasher()
	if err != nil {
		sendError(errors.Wrap(err, "unable to create hasher"))
		return
	}

	// Create a ticker to trigger polling at regular intervals. Ensure that it's
	// cancelled when we're done.
	ticker := time.NewTicker(scanPollInterval)
	defer ticker.Stop()

	// Set up a cancellable watch and ensure that it's cancelled when this
	// handler exits. We don't monitor for watch failure, because it might fail
	// in perfectly reasonable circumstances (e.g. the path not existing). In
	// that case we have to fall back to polling.
	context, cancel := context.WithCancel(context.Background())
	watchEvents := make(chan struct{}, watchEventsBufferSize)
	go watch(context, e.root, watchEvents)
	defer cancel()

	// Create a Goroutine that'll monitor for force requests. It will die once
	// the stream is closed, which will happen automatically once the handler
	// returns. If it fails before receiving a force request, it closes the
	// forces channel (in which case the loop should abort because something is
	// wrong with the stream), otherwise it sends an empty value (in which case
	// the loop should force the response).
	forces := make(chan struct{}, 1)
	go func() {
		var forceRequest scanRequest
		if stream.Decode(&forceRequest) != nil {
			close(forces)
		} else {
			forces <- struct{}{}
		}
	}()

	// Loop until we're done.
	forced := false
	for {
		// Create a snapshot.
		snapshot, cache, err := sync.Scan(e.root, hasher, e.cache, e.ignores)
		if err != nil {
			sendError(errors.Wrap(err, "unable to create snapshot"))
			return
		}

		// Store the cache.
		if err := encoding.MarshalAndSaveProtobuf(e.cachePath, cache); err != nil {
			sendError(errors.Wrap(err, "unable to save cache"))
			return
		}
		e.cache = cache

		// Marshal the snapshot.
		snapshotBytes, err := snapshot.Encode()
		if err != nil {
			sendError(errors.Wrap(err, "unable to marshal snapshot"))
			return
		}

		// If we've been forced or the checksum differs, send the snapshot.
		if forced || !snapshotChecksumMatch(snapshotBytes, request.ExpectedSnapshotChecksum) {
			// Compute the delta.
			delta, err := deltafySnapshot(snapshotBytes, request.BaseSnapshotSignature)
			if err != nil {
				sendError(errors.Wrap(err, "unable to deltafy snapshot"))
				return
			}

			// Done. There's no point in checking for transmission failure
			// because we won't be able to transmit any error.
			stream.Encode(scanResponse{SnapshotDelta: delta})
			return
		}

		// Otherwise, wait until an event occurs that makes us re-scan.
		select {
		case <-ticker.C:
		case <-watchEvents:
		case _, ok := <-forces:
			if !ok {
				sendError(errors.New("error waiting for force request"))
				return
			}
			forced = true
		}
	}
}

func (e *endpoint) transmit(stream *rpc.HandlerStream) {
	// Create an error transmitter.
	sendError := func(err error) {
		stream.Encode(transmitResponse{Error: err.Error()})
	}

	// Receive the request.
	var request transmitRequest
	if err := stream.Decode(&request); err != nil {
		sendError(errors.Wrap(err, "unable to decode request"))
		return
	}

	// Lock the endpoint for reading and defer its release.
	e.RLock()
	defer e.RUnlock()

	// If we're not initialized, we can't do anything.
	if e.version == Version_Unknown {
		sendError(errors.New("endpoint not initialized"))
		return
	}

	// Open the file and ensure it's closed when we're done.
	file, err := os.Open(filepath.Join(e.root, request.Path))
	if err != nil {
		sendError(errors.Wrap(err, "unable to open source file"))
		return
	}
	defer file.Close()

	// Create an rsyncer.
	rsyncer := rsync.New()

	// Create an operation transmitter.
	transmit := func(operation rsync.Operation) error {
		return stream.Encode(transmitResponse{Operation: operation})
	}

	// Transmit the delta. We signal the end of the stream by closing it, and
	// returning from the handler will do that by default.
	if err := rsyncer.Deltafy(file, request.BaseSignature, transmit); err != nil {
		sendError(errors.Wrap(err, "unable to transmit delta"))
		return
	}
}

func (e *endpoint) wipeStaging() error {
	// List the contents in the staging directory.
	contents, err := filesystem.DirectoryContents(e.stagingPath)
	if err != nil {
		return errors.Wrap(err, "unable to list staging directory contents")
	}

	// Remove each of them. Abort if there's a failure.
	for _, name := range contents {
		if err := os.Remove(filepath.Join(e.stagingPath, name)); err != nil {
			return errors.Wrap(err, "unable to remove file")
		}
	}

	// Success.
	return nil
}

func (e *endpoint) Provide(path string, entry *sync.Entry) (string, error) {
	// Compute the expected staging name. This doesn't need to be stable, we can
	// change it in the future, it only needs to be stable across a single
	// synchronization cycle.
	return filepath.Join(e.stagingPath, fmt.Sprintf("%x_%t_%x",
		sha1.Sum([]byte(path)), entry.Executable, entry.Digest,
	)), nil
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
	transmission *rpc.ClientStream
}

func (e *endpoint) dispatch(
	context context.Context,
	queued <-chan stagingOperation,
	dispatched chan<- stagingOperation,
) error {
	// Create an rsyncer.
	rsyncer := rsync.New()

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
		baseSignature, err := rsyncer.Signature(operation.base)
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
		if err := transmission.Encode(request); err != nil {
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
	// Create an rsyncer.
	rsyncer := rsync.New()

	// Track the progress index.
	index := uint64(0)

	// Loop until we're out of operations or cancelled.
	for {
		// Grab the next operation.
		var path string
		var entry *sync.Entry
		var base readSeekCloser
		var transmission *rpc.ClientStream
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
		temporary, err := ioutil.TempFile(e.stagingPath, "staging")
		if err != nil {
			base.Close()
			transmission.Close()
			return errors.Wrap(err, "unable to create temporary file")
		}
		temporaryPath := temporary.Name()

		// Create an rsync operation receiver. Note that this needs to pass back
		// an io.EOF that it receives directly in order to inform the rsyncer
		// that operations are complete, so we don't want to wrap this error.
		receive := func() (rsync.Operation, error) {
			var response transmitResponse
			if err := transmission.Decode(&response); err != nil {
				return rsync.Operation{}, err
			} else if response.Error != "" {
				return rsync.Operation{}, errors.Wrap(
					errors.New(response.Error),
					"transmission error",
				)
			}
			return response.Operation, nil
		}

		// Create a verification hasher.
		hasher, err := e.version.hasher()
		if err != nil {
			base.Close()
			transmission.Close()
			return errors.Wrap(err, "unable to create hasher")
		}

		// Apply patch operations.
		err = rsyncer.Patch(temporary, base, receive, hasher)

		// Close files.
		temporary.Close()
		base.Close()

		// Close the transmission stream.
		transmission.Close()

		// If there was a receiving error, remove the file. We don't abort the
		// staging pipeline in this case, because it's possible that the base
		// was being concurrently modified and that future receives won't fail.
		if err != nil {
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
		stagingPath, err := e.Provide(path, entry)
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

func (e *endpoint) stage(stream *rpc.HandlerStream) {
	// Create an error transmitter.
	sendError := func(err error) {
		stream.Encode(stageResponse{Error: err.Error()})
	}

	// Receive the request.
	var request stageRequest
	if err := stream.Decode(&request); err != nil {
		sendError(errors.Wrap(err, "unable to receive request"))
		return
	}

	// Lock the endpoint and defer its release.
	e.RLock()
	defer e.RUnlock()

	// If we're not initialized, we can't do anything.
	if e.version == Version_Unknown {
		sendError(errors.New("endpoint not initialized"))
		return
	}

	// Compute the staging operations that we'll need to perform.
	operations, err := sync.StagingOperationsForChanges(request.Transitions)
	if err != nil {
		sendError(errors.Wrap(err, "unable to compute staging operations"))
		return
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
		return stream.Encode(stageResponse{Status: s})
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
		sendError(pipelineError)
		return
	}

	// Success.
	stream.Encode(stageResponse{Done: true})
}

func (e *endpoint) apply(stream *rpc.HandlerStream) {
	// Create an error transmitter.
	sendError := func(err error) {
		stream.Encode(applyResponse{Error: err.Error()})
	}

	// Receive the request.
	var request applyRequest
	if err := stream.Decode(&request); err != nil {
		sendError(errors.Wrap(err, "unable to receive request"))
		return
	}

	// Lock the endpoint and defer its release.
	e.Lock()
	defer e.Unlock()

	// If we're not initialized, we can't do anything.
	if e.version == Version_Unknown {
		sendError(errors.New("endpoint not initialized"))
		return
	}

	// Perform application.
	changes, problems := sync.Transition(e.root, request.Transitions, e.cache, e)

	// Wipe the staging directory. We ignores any errors here because we need to
	// send back our transition results at this point. If there's some sort of
	// disk error, it'll be caught by the next round of staging.
	e.wipeStaging()

	// Send the final response.
	stream.Encode(applyResponse{
		Changes:  changes,
		Problems: problems,
	})
}
