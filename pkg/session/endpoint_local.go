package session

import (
	"context"
	"hash"
	"io"
	"os"
	"path/filepath"
	syncpkg "sync"

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
	// scanParametersLock serializes access to the scan-related fields below
	// (those that are updated during scans). Even though we enforce that an
	// endpoint's scan method can't be called concurrently, we perform
	// asynchronous cache disk writes, and thus we need to be sure that we don't
	// re-enter scan and start mutating the following fields while the write
	// Goroutine is still running. We also acquire this lock during transitions
	// since they re-use scan parameters.
	scanParametersLock syncpkg.Mutex
	// cacheWriteError is the last error encountered when trying to write the
	// cache to disk, if any.
	cacheWriteError error
	// cache is the cache from the last successful scan on the endpoint.
	cache *sync.Cache
	// ignoreCache is the ignore cache from the last successful scan on the
	// endpoint.
	ignoreCache sync.IgnoreCache
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
	// Grab the scan lock.
	e.scanParametersLock.Lock()

	// Check for asynchronous cache write errors. If we've encountered one, we
	// don't proceed. Note that we use a defer to unlock since we're grabbing
	// the cacheWriteError on the next line (this avoids an intermediate
	// assignment).
	if e.cacheWriteError != nil {
		defer e.scanParametersLock.Unlock()
		return nil, false, errors.Wrap(e.cacheWriteError, "unable to save cache to disk"), false
	}

	// Perform the scan. If there's an error, we have to assume it's a
	// concurrent modification and just suggest a retry.
	result, preservesExecutability, recomposeUnicode, newCache, newIgnoreCache, err := sync.Scan(
		e.root, e.scanHasher, e.cache, e.ignores, e.ignoreCache, e.symlinkMode,
	)
	if err != nil {
		e.scanParametersLock.Unlock()
		return nil, false, err, true
	}

	// Store the cache, ignore cache, and recommended Unicode recomposition
	// behavior.
	e.cache = newCache
	e.ignoreCache = newIgnoreCache
	e.recomposeUnicode = recomposeUnicode

	// Save the cache to disk in a background Goroutine, allowing this Goroutine
	// to unlock the scan lock once the write is complete.
	go func() {
		if err := encoding.MarshalAndSaveProtobuf(e.cachePath, e.cache); err != nil {
			e.cacheWriteError = err
		}
		e.scanParametersLock.Unlock()
	}()

	// Done.
	return result, preservesExecutability, nil, false
}

func (e *localEndpoint) stageFromRoot(path string, digest []byte, reverseLookupMap *sync.ReverseLookupMap) bool {
	// See if we can find a path within the root that has a matching digest.
	sourcePath, sourcePathOk := reverseLookupMap.Lookup(digest)
	if !sourcePathOk {
		return false
	}

	// Open the source file and defer its closure.
	source, err := os.Open(filepath.Join(e.root, sourcePath))
	if err != nil {
		return false
	}
	defer source.Close()

	// Create a staging sink. We explicitly manage its closure below.
	sink, err := e.stager.Sink(path)
	if err != nil {
		return false
	}

	// Copy data to the sink and close it, then check for copy errors.
	_, err = io.Copy(sink, source)
	sink.Close()
	if err != nil {
		return false
	}

	// Ensure that everything staged correctly.
	_, err = e.stager.Provide(path, digest)
	return err == nil
}

func (e *localEndpoint) stage(entries map[string]*sync.Entry) ([]string, []rsync.Signature, rsync.Receiver, error) {
	// It's possible that a previous staging was interrupted, so look for paths
	// that are already staged by checking if our staging coordinator can
	// already provide them. If everything was already staged, then we can abort
	// the staging operation.
	for path, entry := range entries {
		if _, err := e.stager.Provide(path, entry.Digest); err == nil {
			delete(entries, path)
		}
	}
	if len(entries) == 0 {
		return nil, nil, nil, nil
	}

	// It's possible that we're dealing with renames or copies, so generate a
	// reverse lookup map from the cache and see if we can find any files
	// locally. If we manage to handle all files, then we can abort the staging
	// operation.
	e.scanParametersLock.Lock()
	reverseLookupMap, err := e.cache.GenerateReverseLookupMap()
	e.scanParametersLock.Unlock()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to generate reverse lookup map")
	}
	for path, entry := range entries {
		if e.stageFromRoot(path, entry.Digest, reverseLookupMap) {
			delete(entries, path)
		}
	}
	if len(entries) == 0 {
		return nil, nil, nil, nil
	}

	// Create an rsync engine.
	engine := rsync.NewEngine()

	// Extract paths.
	paths := make([]string, 0, len(entries))
	for path := range entries {
		paths = append(paths, path)
	}

	// Compute signatures for each of the unstaged paths. For paths that don't
	// exist or that can't be read, just use an empty signature, which means to
	// expect/use an empty base when deltafying/patching.
	signatures := make([]rsync.Signature, len(paths))
	for i, path := range paths {
		if base, err := os.Open(filepath.Join(e.root, path)); err != nil {
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
	receiver, err := rsync.NewReceiver(e.root, paths, signatures, e.stager)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to create rsync receiver")
	}

	// Done.
	return paths, signatures, receiver, nil
}

func (e *localEndpoint) supply(paths []string, signatures []rsync.Signature, receiver rsync.Receiver) error {
	return rsync.Transmit(e.root, paths, signatures, receiver)
}

func (e *localEndpoint) transition(transitions []*sync.Change) ([]*sync.Entry, []*sync.Problem, error) {
	// Lock and defer release of the scan parameters lock.
	e.scanParametersLock.Lock()
	defer e.scanParametersLock.Unlock()

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
