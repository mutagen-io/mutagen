package session

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/net/context"

	"github.com/pkg/errors"

	"github.com/rjeczalik/notify"

	"github.com/havoc-io/mutagen/encoding"
	"github.com/havoc-io/mutagen/filesystem"
	"github.com/havoc-io/mutagen/rsync"
	"github.com/havoc-io/mutagen/sync"
)

const (
	alphaCacheName = "alpha_cache"
	betaCacheName  = "beta_cache"
)

type Endpoint struct {
	syncpkg.RWMutex
	version         SessionVersion
	root            string
	caseInsensitive bool
	cachePath       string
	stagingPath     string
}

func NewEndpoint() *Endpoint {
	return &Endpoint{}
}

func (e *Endpoint) Initialize(_ context.Context, request *InitializeRequest) (*InitializeResponse, error) {
	// Lock the endpoint and ensure its release.
	e.Lock()
	defer e.Unlock()

	// If we're already initialized, we can't do it again.
	if e.version != SessionVersion_Unknown {
		return nil, errors.New("endpoint already initialized")
	}

	// Validate the request.
	if request.Session == "" {
		return nil, errors.New("empty session identifier")
	} else if !request.Version.supported() {
		return nil, errors.New("unsupported session version")
	} else if request.Root == "" {
		return nil, errors.New("empty root path")
	}

	// Determine whether or not the root is case-sensitive. This will also
	// ensure that the endpoint has write access to this path.
	caseInsensitive, err := filesystem.CaseInsensitive(request.Root)
	if err != nil {
		return nil, errors.Wrap(err, "unable to determine case-sensitivity")
	}

	// Compute the cache path.
	cacheName := alphaCacheName
	if !request.Alpha {
		cacheName = betaCacheName
	}
	cachePath, err := subpath(request.Session, cacheName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute cache path")
	}

	// Compute (and create) the staging path.
	stagingPath, err := stagingPath(request.Session)
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute staging path")
	}

	// Check if the root decomposes Unicode.
	decomposesUnicode, err := filesystem.DecomposesUnicode(request.Root)
	if err != nil {
		return nil, errors.Wrap(err, "unable to determine Unicode decomposition behavior")
	}

	// At the moment, we don't do any filesystem checks for executability
	// preservation, because there's not really a meaningful way to test this on
	// the filesystem and it falls on OS boundaries anyway. We just use a
	// hard-coded constant.

	// Store parameters, thereby marking the endpoint as initialized.
	e.version = request.Version
	e.root = request.Root
	e.caseInsensitive = caseInsensitive
	e.cachePath = cachePath
	e.stagingPath = stagingPath

	// Success.
	return &InitializeResponse{
		DecomposesUnicode:      decomposesUnicode,
		PreservesExecutability: preservesExecutability,
	}, nil
}

const (
	watchBufferSize = 10
)

func (e *Endpoint) Watch(request *WatchRequest, responses Endpoint_WatchServer) error {
	// Read-lock the endpoint and ensure its release.
	e.RLock()
	defer e.RUnlock()

	// If we're not initialized, we can't do anything.
	if e.version == SessionVersion_Unknown {
		return nil, errors.New("endpoint not initialized")
	}

	// Create a channel to receive watch events. It needs to be buffered because
	// the watcher sends events in a non-blocking fashion and we might not be
	// able to transmit notifications quite as fast as they can be generated.
	events := make(chan notify.EventInfo, watchBufferSize)

	// Set up the watcher and make sure it shuts down when this handler
	// terminates
	if err := notify.Watch(e.root+"/...", events, notify.All); err != nil {
		return errors.Wrap(err, "unable to create watch")
	}
	defer notify.Stop(events)

	// Grab the cancellation channel.
	done := responses.Context().Done()

	// Create a response we can re-use.
	response := &WatchResponse{}

	// Wait for events or termination.
	for {
		select {
		case <-events:
			if err := responses.Send(response); err != nil {
				return errors.Wrap(err, "unable to transmit notification")
			}
		case <-done:
			break
		}
	}
}

func (e *Endpoint) Stage(stream Endpoint_StageServer) error {
	// Read-lock the endpoint and ensure its release.
	e.RLock()
	defer e.RUnlock()

	// If we're not initialized, we can't do anything.
	if e.version == SessionVersion_Unknown {
		return nil, errors.New("endpoint not initialized")
	}

	// Grab the initial request.
	request, err := stream.Recv()
	if err != nil {
		return errors.Wrap(err, "unable to receive request")
	}

	// Compute the path where this file will be staged. We base this on target
	// path, executability, and the expected digest (which we verify before
	// staging). This format doesn't need to be stable, because the staging
	// directory is wiped during application anyway.
	stagingPath := filepath.Join(e.stagingPath, fmt.Sprintf(
		"%x-%t-%x",
		sha1.Sum([]byte(request.Path)),
		request.Executable,
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
	output, err := ioutil.TempFile(e.stagingPath, "incoming")
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

	// Set the target file mode.
	if request.Executable {
		err = os.Chmod(outputName, 0700)
	} else {
		err = os.Chmod(outputName, 0600)
	}
	if err != nil {
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
	// Read-lock the endpoint and ensure its release.
	e.RLock()
	defer e.RUnlock()

	// If we're not initialized, we can't do anything.
	if e.version == SessionVersion_Unknown {
		return nil, errors.New("endpoint not initialized")
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

func (e *Endpoint) Snapshot(_ context.Context, request *SnapshotRequest) (*SnapshotResponse, error) {
	// Read-lock the endpoint and ensure its release.
	e.RLock()
	defer e.RUnlock()

	// If we're not initialized, we can't do anything.
	if e.version == SessionVersion_Unknown {
		return nil, errors.New("endpoint not initialized")
	}

	// Load the cache. If it fails, just create an empty cache.
	cache := &sync.Cache{}
	if encoding.LoadAndUnmarshalProtobuf(e.cachePath, cache) != nil {
		*cache = sync.Cache{}
	}

	// Create a hasher.
	hasher, err := e.version.hasher()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create hasher")
	}

	// Perform the snapshot.
	snapshot, cache, err := sync.Snapshot(e.root, hasher, cache)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create snapshot")
	}

	// Attempt to save the new cache. Technically we can ignore errors here, but
	// they are probably indicative of something else, so we should try not to
	// ignore them.
	if err := encoding.MarshalAndSaveProtobuf(e.cachePath, cache); err != nil {
		return nil, errors.Wrap(err, "unable to save cache")
	}

	// Serialize the snapshot and make it an io.Reader.
	serialized, err := snapshot.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "unable to serialize snapshot")
	}
	reader := bytes.NewReader(serialized)

	// Create a container for delta operations and a writer to fill it. It's
	// very important to note that the Deltafy method re-uses operations and
	// their data buffers, so we have to make a copy when retaining them in this
	// slice.
	var operations []*rsync.Operation
	writer := func(o *rsync.Operation) error {
		data := make([]byte, len(o.Data))
		copy(data, o.Data)
		operations = append(operations, &rsync.Operation{
			Type:          o.Type,
			BlockIndex:    o.BlockIndex,
			BlockIndexEnd: o.BlockIndexEnd,
			Data:          data,
		})
		return nil
	}

	// Create an rsyncer and perform deltafication.
	rsyncer := rsync.New()
	if err := rsyncer.Deltafy(reader, request.BaseSignature, writer); err != nil {
		return nil, errors.Wrap(err, "unable to perform deltafication")
	}

	// Success.
	return &SnapshotResponse{Operations: operations}, nil
}

// TODO: Add Apply.
