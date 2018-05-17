package session

import (
	"context"
	"hash"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/encoding"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/sync"
)

type localEndpoint struct {
	// root is the synchronization root for the endpoint. It is static.
	root string
	// watchCancel cancels filesystem monitoring. It is static.
	watchCancel context.CancelFunc
	// watchEvents is the filesystem monitoring channel. It is static.
	watchEvents chan struct{}
	// ignores is the list of ignored paths for the session. It is static.
	ignores []string
	// cachePath is the path at which to save the cache for the session. It is
	// static.
	cachePath string
	// cache is the cache from the last successful scan on the endpoint.
	cache *sync.Cache
	// scanHasher is the hasher used for scans.
	scanHasher hash.Hash
	// stager is the staging coordinator.
	stager *stager
}

func newLocalEndpoint(session string, version Version, root string, ignores []string, alpha bool) (endpoint, error) {
	// Validate endpoint parameters.
	if session == "" {
		return nil, errors.New("empty session identifier")
	} else if !version.supported() {
		return nil, errors.New("unsupported session version")
	} else if root == "" {
		return nil, errors.New("empty root path")
	}

	// Expand and normalize the root path.
	root, err := filesystem.Normalize(root)
	if err != nil {
		return nil, errors.Wrap(err, "unable to normalize root path")
	}

	// Start file monitoring for the root.
	watchContext, watchCancel := context.WithCancel(context.Background())
	watchEvents := make(chan struct{}, 1)
	go filesystem.Watch(watchContext, root, watchEvents)

	// Compute the cache path.
	cachePath, err := pathForCache(session, alpha)
	if err != nil {
		watchCancel()
		return nil, errors.Wrap(err, "unable to compute/create cache path")
	}

	// Load any existing cache. If it fails to load or validate, just replace it
	// with an empty one.
	// TODO: Should we let validation errors bubble up? They may be indicative
	// of something bad.
	cache := &sync.Cache{}
	if encoding.LoadAndUnmarshalProtobuf(cachePath, cache) != nil {
		cache = &sync.Cache{}
	} else if cache.EnsureValid() != nil {
		cache = &sync.Cache{}
	}

	// Create a staging coordinator.
	stager, err := newStager(session, version, alpha)
	if err != nil {
		watchCancel()
		return nil, errors.Wrap(err, "unable to create staging coordinator")
	}

	// Success.
	return &localEndpoint{
		root:        root,
		watchCancel: watchCancel,
		watchEvents: watchEvents,
		ignores:     ignores,
		cachePath:   cachePath,
		cache:       cache,
		scanHasher:  version.hasher(),
		stager:      stager,
	}, nil
}

func (e *localEndpoint) poll(context context.Context) error {
	// Wait for either cancellation or an event.
	select {
	case _, ok := <-e.watchEvents:
		if !ok {
			return errors.New("endpoint watcher terminated")
		}
	case <-context.Done():
	}

	// Done.
	return nil
}

func (e *localEndpoint) scan(ancestor *sync.Entry) (*sync.Entry, bool, error) {
	// Perform the scan. If there's an error, we have to assume it's a
	// concurrent modification and just suggest a retry.
	result, newCache, err := sync.Scan(e.root, e.scanHasher, e.cache, e.ignores)
	if err != nil {
		return nil, true, nil
	}

	// Propagate executability from the ancestor to the result if necessary.
	if !filesystem.PreservesExecutability {
		result = sync.PropagateExecutability(ancestor, result)
	}

	// Store the cache.
	e.cache = newCache
	if err = encoding.MarshalAndSaveProtobuf(e.cachePath, e.cache); err != nil {
		return nil, false, errors.Wrap(err, "unable to save cache")
	}

	// Done.
	return result, false, nil
}

func (e *localEndpoint) stage(paths []string, entries []*sync.Entry) ([]string, []rsync.Signature, rsync.Receiver, error) {
	// It's possible that a previous staging was interrupted, so look for paths
	// that are already staged by checking if our staging coordinator can
	// already provide them.
	unstagedPaths := make([]string, 0, len(paths))
	for i, p := range paths {
		if _, err := e.stager.Provide(p, entries[i], 0); err != nil {
			unstagedPaths = append(unstagedPaths, p)
		}
	}

	// Create an rsync engine.
	engine := rsync.NewEngine()

	// Compute signatures for each of the unstaged paths. For paths that don't
	// exist or that can't be read, just use an empty signature, which means to
	// expect/use an empty base when deltafying/patching.
	signatures := make([]rsync.Signature, len(unstagedPaths))
	for i, p := range unstagedPaths {
		if base, err := os.Open(filepath.Join(e.root, p)); err != nil {
			continue
		} else if signature, err := engine.Signature(base, 0); err != nil {
			base.Close()
			continue
		} else {
			base.Close()
			signatures[i] = signature
		}
	}

	// Create a receiver.
	receiver, err := rsync.NewReceiver(e.root, unstagedPaths, signatures, e.stager)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to create rsync receiver")
	}

	// Done.
	return unstagedPaths, signatures, receiver, nil
}

func (e *localEndpoint) supply(paths []string, signatures []rsync.Signature, receiver rsync.Receiver) error {
	return rsync.Transmit(e.root, paths, signatures, receiver)
}

func (e *localEndpoint) transition(transitions []*sync.Change) ([]*sync.Change, []*sync.Problem, error) {
	// Perform the transition.
	changes, problems := sync.Transition(e.root, transitions, e.cache, e.stager)

	// Wipe the staging directory. We don't monitor for errors here, because we
	// need to return the changes and problems no matter what, but if there's
	// something weird going on with the filesystem, we'll see it the next time
	// we scan or stage.
	// TODO: If we see a large number of problems, should we avoid wiping the
	// staging directory? It could be due to a parent path component missing,
	// which could be corrected.
	e.stager.wipe()

	// Done.
	return changes, problems, nil
}

func (e *localEndpoint) shutdown() error {
	// Terminate filesystem watching. This will result in the associated events
	// channel being closed.
	e.watchCancel()

	// Done.
	return nil
}
