package session

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	syncpkg "sync"

	"golang.org/x/net/context"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/encoding"
	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/sync"
)

type Endpoint struct {
	syncpkg.RWMutex
	version              SessionVersion
	root                 string
	replicaPath          string
	replica              *Archive
	cachePath            string
	cache                *sync.Cache
	stagingDirectoryPath string
}

func NewEndpoint() *Endpoint {
	return &Endpoint{}
}

func (e *Endpoint) Initialize(_ context.Context, request *InitializeRequest) (*InitializeResponse, error) {
	// Lock the endpoint for modification and ensure its release.
	e.Lock()
	defer e.Unlock()

	// If we're already initialized, we can't do it again.
	if e.version != SessionVersion_Unknown {
		return nil, errors.New("endpoint already initialized")
	}

	// Validate the request. Other parameters, such as the archive checksum and
	// archive, will be validated below.
	if request.Session == "" {
		return nil, errors.New("empty session identifier")
	} else if !request.Version.supported() {
		return nil, errors.New("unsupported session version")
	} else if request.Root == "" {
		return nil, errors.New("empty root path")
	}

	// TODO: Perform tilde-expansion and resolution on the root path.
	root := request.Root

	// Compute the replica path.
	replicaPath, err := pathForReplica(request.Session, request.Alpha)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute replica path")
	}

	// If an archive has been provided, use that as our replica.
	replica := request.Archive

	// If no archive was provided, load ours from disk. If this fails, just
	// create an empty one.
	if replica == nil {
		replica = &Archive{}
		if encoding.LoadAndUnmarshalProtobuf(replicaPath, replica) != nil {
			*replica = Archive{}
		}
	}

	// Verify that the replica checksum matches what's expected. We only really
	// need to do this when loading from disk, but it doesn't cost much to
	// verify in every case. If there's a mismatch, report it in a way that the
	// controller can detect (don't just throw an error, because we can't really
	// send sentinel errors across gRPC boundaries).
	if replicaChecksum, err := checksum(replica.Root); err != nil {
		return nil, errors.Wrap(err, "unable to compute replica checksum")
	} else if !bytes.Equal(replicaChecksum, request.ArchiveChecksum) {
		return &InitializeResponse{ReplicaChecksumMismatch: true}, nil
	}

	// Compute the cache path.
	cachePath, err := pathForCache(request.Session, request.Alpha)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute cache path")
	}

	// Load the cache. If it fails, just create an empty cache.
	cache := &sync.Cache{}
	if encoding.LoadAndUnmarshalProtobuf(cachePath, cache) != nil {
		*cache = sync.Cache{}
	}

	// Compute (and create) the staging path.
	stagingDirectoryPath, err := pathForStaging(request.Session, request.Alpha)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute staging path")
	}

	// Store parameters, thereby marking the endpoint as initialized.
	e.version = request.Version
	e.root = root
	e.replicaPath = replicaPath
	e.replica = replica
	e.cachePath = cachePath
	e.cache = cache
	e.stagingDirectoryPath = stagingDirectoryPath

	// Success.
	return &InitializeResponse{}, nil
}

const (
	scanEventBufferSize = 10
)

func (e *Endpoint) Scan(ctx context.Context, request *ScanRequest) (*ScanResponse, error) {
	// Lock the endpoint for modification and ensure its release.
	e.Lock()
	defer e.Unlock()

	// If we're not initialized, we can't do anything.
	if e.version == SessionVersion_Unknown {
		return nil, errors.New("endpoint not initialized")
	}

	// Verify that the replica checksum matches what's expected. If we trust our
	// entry algebra, then we only really need to do this when loading from disk
	// during initialization (because the endpoint could have terminated while
	// saving the replica), but it doesn't cost much to verify in every case and
	// gives us a sanity check that our algebra works correctly. If there's a
	// mismatch in here, then there's a problem with the code and we should
	// abort.
	if replicaChecksum, err := checksum(e.replica.Root); err != nil {
		return nil, errors.Wrap(err, "unable to compute replica checksum")
	} else if !bytes.Equal(replicaChecksum, request.ArchiveChecksum) {
		return nil, errors.New("replica checksum mismatch")
	}

	// Create a hasher.
	hasher, err := e.version.hasher()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create hasher")
	}

	// Create a watch context and ensure that it is cancelled by the time we
	// leave this handler.
	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()

	// Create a watch events channel. We create it buffered because the watch
	// routines send events in a non-blocking fashion. We're probably
	// over-buffering this a bit, because for native watching we're coallescing
	// events and almost sure to see only mutating events (thus exiting the loop
	// after the first event) and for polling watching we have a watch period
	// that'll be much larger than the time it takes to create a snapshot. In
	// any case, this over-buffering isn't expensive and keeps us safe. If we
	// happen to under-buffer, we'll just catch the next timer-based event.
	watchEvents := make(chan struct{}, scanEventBufferSize)

	// Start watching, monitoring for errors.
	watchErrors := make(chan error, 1)
	go func() {
		watchErrors <- watch(watchCtx, e.root, watchEvents)
	}()

	// Create snapshots until we find one that differs from expected, using the
	// watcher to regulate our snapshotting.
	for {
		// Perform a snapshot.
		snapshot, cache, err := sync.Scan(e.root, hasher, e.cache)
		if err != nil {
			return nil, errors.Wrap(err, "unable to create snapshot")
		}

		// Compute the snapshot checksum.
		snapshotChecksum, err := checksum(snapshot)
		if err != nil {
			return nil, errors.Wrap(err, "unable to compute snapshot checksum")
		}

		// If it differs from expected, we're done.
		if !bytes.Equal(snapshotChecksum, request.ExpectedSnapshotChecksum) {
			// Save and record the cache.
			if err := encoding.MarshalAndSaveProtobuf(e.cachePath, cache); err != nil {
				return nil, errors.Wrap(err, "unable to save cache")
			}
			e.cache = cache

			// Return a delta.
			return &ScanResponse{
				Delta: sync.Diff(e.replica.Root, snapshot),
			}, nil
		}

		// Otherwise, wait until something changes.
		select {
		case <-ctx.Done():
			return nil, errors.New("scan cancelled")
		case err := <-watchErrors:
			return nil, errors.Wrap(err, "watch error")
		case <-watchEvents:
		}
	}
}

func (e *Endpoint) Stage(stream Endpoint_StageServer) error {
	// Lock the endpoint for reading (since we'll probably have concurrent
	// staging operations) and ensure its release.
	e.RLock()
	defer e.RUnlock()

	// If we're not initialized, we can't do anything.
	if e.version == SessionVersion_Unknown {
		return errors.New("endpoint not initialized")
	}

	// Grab the initial request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Compute the path where this file will be staged.
	stagingPath := filepath.Join(e.stagingDirectoryPath, nameForStaging(
		request.Path,
		request.Digest,
	))

	// Check if we already staged this during a previous synchronization session
	// that was interrupted. If so, notify the client and bail.
	if _, err := os.Lstat(stagingPath); err == nil {
		if err = stream.Send(&StageResponse{AlreadyStaged: true}); err != nil {
			return errors.Wrap(err, "unable to notify of previous staging")
		}
		return nil
	}

	// Attempt to open the base. If there is an error (which there may well be
	// if this is a creation and the base doesn't exist), then don't worry, we
	// can use an empty base.
	var base io.ReadSeeker
	if f, err := os.Open(filepath.Join(e.root, request.Path)); err != nil {
		base = &bytes.Reader{}
	} else {
		base = f
		defer f.Close()
	}

	// Create an rsyncer.
	rsyncer := rsync.New()

	// Compute the base signature. If there's a failure here, then we can still
	// recover by using an empty signature and base.
	signature, err := rsyncer.Signature(base)
	if err != nil {
		base = &bytes.Reader{}
		signature = nil
	}

	// Send the signature response.
	if err := stream.Send(&StageResponse{BaseSignature: signature}); err != nil {
		return errors.Wrap(err, "unable to send signature response")
	}

	// Open a temporary staging file and compute its name. We don't defer its
	// closure because we need to rename it.
	output, err := ioutil.TempFile(e.stagingDirectoryPath, "incoming")
	if err != nil {
		return errors.Wrap(err, "unable to create output")
	}
	outputName := output.Name()

	// Create a wrapper to read incoming operations from the message stream. We
	// treat an EOF as an indication that we're done, and we signal that to
	// rsync by passing a nil operation.
	reader := func() (*rsync.Operation, error) {
		request, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil, nil
			}
			return nil, err
		}
		return request.Operation, nil
	}

	// Create a hasher.
	hasher, err := e.version.hasher()
	if err != nil {
		return errors.Wrap(err, "unable to create hasher")
	}

	// Receive and process the deltafied file.
	if err := rsyncer.Patch(output, base, reader, hasher); err != nil {
		output.Close()
		return errors.Wrap(err, "unable to patch base")
	}

	// Close the output.
	output.Close()

	// Compute the output digest and ensure it matches what's expected. We could
	// easily run into cases where concurrent modifications cause this to happen
	// without triggering full errors. There's not much point in notifying the
	// client about this (all it could do is restage, but if it did that it'd
	// probably see the same discrepancy anyway since something was modified),
	// so we just remove and move on, the application will simply fail.
	digest := hasher.Sum(nil)
	if !bytes.Equal(digest, request.Digest) {
		os.Remove(outputName)
		return errors.New("patched digest did not match expected")
	}

	// TODO: Set the target file mode.
	if true {
		os.Remove(outputName)
		return errors.Wrap(err, "unable to set file mode")
	}

	// Rename the staging file.
	if err := os.Rename(outputName, stagingPath); err != nil {
		os.Remove(outputName)
		return errors.Wrap(err, "unable to rename file to staging path")
	}

	// Success.
	return nil
}

func (e *Endpoint) Transmit(request *TransmitRequest, responses Endpoint_TransmitServer) error {
	// Lock the endpoint for reading (since we'll probably have concurrent
	// transmission operations) and ensure its release.
	e.RLock()
	defer e.RUnlock()

	// If we're not initialized, we can't do anything.
	if e.version == SessionVersion_Unknown {
		return errors.New("endpoint not initialized")
	}

	// Create an rsyncer.
	rsyncer := rsync.New()

	// Open the target and ensure its closure.
	target, err := os.Open(filepath.Join(e.root, request.Path))
	if err != nil {
		return errors.Wrap(err, "unable to open target file")
	}
	defer target.Close()

	// Create a wrapper to write operations to the message stream.
	writer := func(operation *rsync.Operation) error {
		return responses.Send(&TransmitResponse{Operation: operation})
	}

	// Perform streaming.
	// TODO: Should we errors.Wrap this? The semantics are almost the same as
	// the surrounding function, but not quite.
	return rsyncer.Deltafy(target, request.BaseSignature, writer)
}

func (e *Endpoint) Apply(_ context.Context, request *ApplyRequest) (*ApplyResponse, error) {
	// Lock the endpoint for modification and ensure its release.
	e.Lock()
	defer e.Unlock()

	// TODO: Implement.
	return nil, errors.New("not implemented")
}

func (e *Endpoint) Update(_ context.Context, request *UpdateRequest) (*UpdateResponse, error) {
	// Lock the endpoint for modification and ensure its release.
	e.Lock()
	defer e.Unlock()

	// Apply the changes to the replica.
	newRoot, err := sync.Apply(e.replica.Root, request.Changes)
	if err != nil {
		return nil, errors.Wrap(err, "unable to apply changes to replica")
	}
	e.replica.Root = newRoot

	// Save the replica.
	if err := encoding.MarshalAndSaveProtobuf(e.replicaPath, e.replica); err != nil {
		return nil, errors.Wrap(err, "unable to save replica")
	}

	// Success.
	return &UpdateResponse{}, nil
}
