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
	// symlinkMode is the symlink mode for the session. It is static.
	symlinkMode sync.SymlinkMode
	// ignores is the list of ignored paths for the session. It is static.
	ignores []string
	// cachePath is the path at which to save the cache for the session. It is
	// static.
	cachePath string
	// cache is the cache from the last successful scan on the endpoint.
	cache *sync.Cache
	// recomposeUnicode is the Unicode recomposition behavior recommended by the
	// last successful scan on the endpoint.
	recomposeUnicode bool
	// scanHasher is the hasher used for scans.
	scanHasher hash.Hash
	// stager is the staging coordinator.
	stager *stager
}

func newLocalEndpoint(session string, version Version, root string, configuration *Configuration, alpha bool) (endpoint, error) {
	// Expand and normalize the root path.
	root, err := filesystem.Normalize(root)
	if err != nil {
		return nil, errors.Wrap(err, "unable to normalize root path")
	}

	// Extract the effective symlink mode.
	symlinkMode := configuration.SymlinkMode
	if symlinkMode == sync.SymlinkMode_SymlinkDefault {
		symlinkMode = version.DefaultSymlinkMode()
	}

	// Extract the effective watch mode.
	watchMode := configuration.WatchMode
	if watchMode == filesystem.WatchMode_WatchDefault {
		watchMode = version.DefaultWatchMode()
	}

	// Extract the effective VCS ignore mode.
	ignoreVCSMode := configuration.IgnoreVCSMode
	if ignoreVCSMode == sync.IgnoreVCSMode_IgnoreVCSDefault {
		ignoreVCSMode = version.DefaultIgnoreVCSMode()
	}

	// Compute a combined ignore list.
	var ignores []string
	if ignoreVCSMode == sync.IgnoreVCSMode_IgnoreVCS {
		ignores = append(ignores, sync.DefaultVCSIgnores...)
	}
	ignores = append(ignores, configuration.DefaultIgnores...)
	ignores = append(ignores, configuration.Ignores...)

	// Start file monitoring for the root.
	watchContext, watchCancel := context.WithCancel(context.Background())
	watchEvents := make(chan struct{}, 1)
	go filesystem.Watch(
		watchContext,
		root,
		watchEvents,
		watchMode,
		configuration.WatchPollingInterval,
	)

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
		symlinkMode: symlinkMode,
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

func (e *localEndpoint) scan(_ *sync.Entry) (*sync.Entry, bool, error, bool) {
	// Perform the scan. If there's an error, we have to assume it's a
	// concurrent modification and just suggest a retry.
	result, preservesExecutability, recomposeUnicode, newCache, err := sync.Scan(e.root, e.scanHasher, e.cache, e.ignores, e.symlinkMode)
	if err != nil {
		return nil, false, err, true
	}

	// Store the cache and recommended Unicode recomposition behavior.
	e.cache = newCache
	e.recomposeUnicode = recomposeUnicode

	// Save the cache to disk.
	if err = encoding.MarshalAndSaveProtobuf(e.cachePath, e.cache); err != nil {
		return nil, false, errors.Wrap(err, "unable to save cache"), false
	}

	// Done.
	return result, preservesExecutability, nil, false
}

func (e *localEndpoint) stage(paths []string, entries []*sync.Entry) ([]string, []rsync.Signature, rsync.Receiver, error) {
	// It's possible that a previous staging was interrupted, so look for paths
	// that are already staged by checking if our staging coordinator can
	// already provide them.
	unstagedPaths := make([]string, 0, len(paths))
	for i, p := range paths {
		if _, err := e.stager.Provide(p, entries[i].Digest); err != nil {
			unstagedPaths = append(unstagedPaths, p)
		}
	}

	// If everything was already staged, then we can abort the staging
	// operation.
	if len(unstagedPaths) == 0 {
		return nil, nil, nil, nil
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

func (e *localEndpoint) transition(transitions []*sync.Change) ([]*sync.Entry, []*sync.Problem, error) {
	// Perform the transition.
	results, problems := sync.Transition(e.root, transitions, e.cache, e.symlinkMode, e.recomposeUnicode, e.stager)

	// Wipe the staging directory. We don't monitor for errors here, because we
	// need to return the results and problems no matter what, but if there's
	// something weird going on with the filesystem, we'll see it the next time
	// we scan or stage.
	// TODO: If we see a large number of problems, should we avoid wiping the
	// staging directory? It could be due to a parent path component missing,
	// which could be corrected.
	e.stager.wipe()

	// Done.
	return results, problems, nil
}

func (e *localEndpoint) shutdown() error {
	// Terminate filesystem watching. This will result in the associated events
	// channel being closed.
	e.watchCancel()

	// Done.
	return nil
}
