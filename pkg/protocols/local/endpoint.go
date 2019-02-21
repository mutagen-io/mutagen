package local

import (
	"context"
	"hash"
	"io"
	syncpkg "sync"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/encoding"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// endpoint provides a local, in-memory implementation of session.Endpoint for
// local files.
type endpoint struct {
	// root is the synchronization root for the endpoint. It is static.
	root string
	// readOnly determines whether or not the endpoint should be operating in a
	// read-only mode (i.e. it is the source of unidirectional synchronization).
	// Although the controller can send the endpoint whatever configuration it
	// wants, there may be a configuration validation option for the endpoint
	// that ensures the endpoint is in a read-only state. In those cases, we
	// want to make sure an untrusted controller can't then try to perform a
	// modification operation. This field is static.
	readOnly bool
	// maximumEntryCount is the maximum number of entries within the
	// synchronization root that this endpoint will support synchronizing. A
	// zero value means that the size is unlimited.
	maximumEntryCount uint64
	// watchCancel cancels filesystem monitoring. It is static.
	watchCancel context.CancelFunc
	// watchEvents is the filesystem monitoring channel. It is static.
	watchEvents chan struct{}
	// symlinkMode is the symlink mode for the session. It is static.
	symlinkMode sync.SymlinkMode
	// ignores is the list of ignored paths for the session. It is static.
	ignores []string
	// defaultFileMode is the default file permission mode to use in "portable"
	// permission propagation. It is static.
	defaultFileMode filesystem.Mode
	// defaultDirectoryMode is the default directory permission mode to use in
	// "portable" permission propagation. It is static.
	defaultDirectoryMode filesystem.Mode
	// defaultOwnership is the default ownership specification to use in
	// "portable" permission propagation. It is static.
	defaultOwnership *filesystem.OwnershipSpecification
	// cachePath is the path at which to save the cache for the session. It is
	// static.
	cachePath string
	// cacheLock locks cacheWriteError and cache. Although endpoint is not
	// designed for concurrent external usage, it performs concurrent cache
	// writes after a scan, which requires that we lock these two parameters
	// (since we use them in other methods that might be called concurrently
	// with the save operation).
	// TODO: Would it make sense to make this a RWMutex? I'm not sure it would
	// help much, because the only time where there'd be contention would be
	// when cache writing wasn't finished by the time staging or transitioning
	// had started, but we'd still need to ensure cacheWriteError was locked for
	// writes by the cache writing Goroutine, so we'd have to use a separate
	// lock for that if we wanted the cache writing Goroutine to only acquire a
	// read lock on the cache. It's just not likely to help much.
	cacheLock syncpkg.Mutex
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
	// lastScanCount is the entry count at the time of the last scan.
	lastScanCount uint64
	// scannedSinceLastStageCall tracks whether or not a scan operation has
	// occurred since the last staging operation.
	scannedSinceLastStageCall bool
	// scannedSinceLastTransitionCall tracks whether or not a scan operation has
	// occurred since the last transitioning operation.
	scannedSinceLastTransitionCall bool
}

// NewEndpoint creates a new local endpoint instance using the specified session
// metadata and options.
func NewEndpoint(
	root,
	sessionIdentifier string,
	version session.Version,
	configuration *session.Configuration,
	alpha bool,
	options ...EndpointOption,
) (session.Endpoint, error) {
	// Expand and normalize the root path.
	root, err := filesystem.Normalize(root)
	if err != nil {
		return nil, errors.Wrap(err, "unable to normalize root path")
	}

	// Create an endpoint configuration and apply all options.
	endpointOptions := &endpointOptions{}
	for _, o := range options {
		o.apply(endpointOptions)
	}

	// Determine if the endpoint is running in a read-only mode.
	synchronizationMode := configuration.SynchronizationMode
	if synchronizationMode.IsDefault() {
		synchronizationMode = version.DefaultSynchronizationMode()
	}
	unidirectional := synchronizationMode == sync.SynchronizationMode_SynchronizationModeOneWaySafe ||
		synchronizationMode == sync.SynchronizationMode_SynchronizationModeOneWayReplica
	readOnly := alpha && unidirectional

	// Compute the effective symlink mode.
	symlinkMode := configuration.SymlinkMode
	if symlinkMode.IsDefault() {
		symlinkMode = version.DefaultSymlinkMode()
	}

	// Compute the effective watch mode.
	watchMode := configuration.WatchMode
	if watchMode.IsDefault() {
		watchMode = version.DefaultWatchMode()
	}

	// Compute the effective watch polling interval.
	watchPollingInterval := configuration.WatchPollingInterval
	if watchPollingInterval == 0 {
		watchPollingInterval = version.DefaultWatchPollingInterval()
	}

	// Compute the effective VCS ignore mode.
	ignoreVCSMode := configuration.IgnoreVCSMode
	if ignoreVCSMode.IsDefault() {
		ignoreVCSMode = version.DefaultIgnoreVCSMode()
	}

	// Compute the effective default file mode.
	defaultFileMode := filesystem.Mode(configuration.DefaultFileMode)
	if defaultFileMode == 0 {
		defaultFileMode = version.DefaultFileMode()
	}

	// Compute the effective default directory mode.
	defaultDirectoryMode := filesystem.Mode(configuration.DefaultDirectoryMode)
	if defaultDirectoryMode == 0 {
		defaultDirectoryMode = version.DefaultDirectoryMode()
	}

	// Compute the effective owner specification.
	defaultOwnerSpecification := configuration.DefaultOwner
	if defaultOwnerSpecification == "" {
		defaultOwnerSpecification = version.DefaultOwnerSpecification()
	}

	// Compute the effective owner group specification.
	defaultGroupSpecification := configuration.DefaultGroup
	if defaultGroupSpecification == "" {
		defaultGroupSpecification = version.DefaultGroupSpecification()
	}

	// Compute the effective ownership specification.
	defaultOwnership, err := filesystem.NewOwnershipSpecification(
		defaultOwnerSpecification,
		defaultGroupSpecification,
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create ownership specification")
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
	if endpointOptions.watchingMechanism != nil {
		go endpointOptions.watchingMechanism(watchContext, root, watchEvents)
	} else {
		go filesystem.Watch(
			watchContext,
			root,
			watchEvents,
			watchMode,
			watchPollingInterval,
		)
	}

	// Compute the cache path.
	var cachePath string
	if endpointOptions.cachePathCallback != nil {
		cachePath, err = endpointOptions.cachePathCallback(sessionIdentifier, alpha)
	} else {
		cachePath, err = pathForCache(sessionIdentifier, alpha)
	}
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

	// Compute the staging root path.
	var stagingRoot string
	if endpointOptions.stagingRootCallback != nil {
		stagingRoot, err = endpointOptions.stagingRootCallback(sessionIdentifier, alpha)
	} else {
		stagingRoot, err = pathForStagingRoot(sessionIdentifier, alpha)
	}
	if err != nil {
		watchCancel()
		return nil, errors.Wrap(err, "unable to compute staging root")
	}

	// Success.
	return &endpoint{
		root:                 root,
		readOnly:             readOnly,
		maximumEntryCount:    configuration.MaximumEntryCount,
		watchCancel:          watchCancel,
		watchEvents:          watchEvents,
		symlinkMode:          symlinkMode,
		ignores:              ignores,
		defaultFileMode:      defaultFileMode,
		defaultDirectoryMode: defaultDirectoryMode,
		defaultOwnership:     defaultOwnership,
		cachePath:            cachePath,
		cache:                cache,
		scanHasher:           version.Hasher(),
		stager:               newStager(version, stagingRoot, configuration.MaximumStagingFileSize),
	}, nil
}

// Poll implements the Poll method for local endpoints.
func (e *endpoint) Poll(context context.Context) error {
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

// Scan implements the Scan method for local endpoints.
func (e *endpoint) Scan(_ *sync.Entry) (*sync.Entry, bool, error, bool) {
	// Grab the cache lock.
	e.cacheLock.Lock()

	// Check for asynchronous cache write errors. If we've encountered one, we
	// don't proceed. Note that we use a defer to unlock since we're grabbing
	// the cacheWriteError on the next line (this avoids an intermediate
	// assignment).
	if e.cacheWriteError != nil {
		defer e.cacheLock.Unlock()
		return nil, false, errors.Wrap(e.cacheWriteError, "unable to save cache to disk"), false
	}

	// Perform the scan. If there's an error, we have to assume it's a
	// concurrent modification and just suggest a retry.
	result, preservesExecutability, recomposeUnicode, newCache, newIgnoreCache, err := sync.Scan(
		e.root, e.scanHasher, e.cache, e.ignores, e.ignoreCache, e.symlinkMode,
	)
	if err != nil {
		e.cacheLock.Unlock()
		return nil, false, err, true
	}

	// Update the last scan count.
	e.lastScanCount = result.Count()

	// Update call states.
	e.scannedSinceLastStageCall = true
	e.scannedSinceLastTransitionCall = true

	// Verify that we haven't exceeded the maximum entry count.
	if e.maximumEntryCount != 0 && e.lastScanCount > e.maximumEntryCount {
		e.cacheLock.Unlock()
		return nil, false, errors.New("exceeded allowed entry count"), true
	}

	// Store the cache, ignore cache, and recommended Unicode recomposition
	// behavior.
	e.cache = newCache
	e.ignoreCache = newIgnoreCache
	e.recomposeUnicode = recomposeUnicode

	// Save the cache to disk in a background Goroutine (and release the cache
	// lock once that's complete).
	go func() {
		if err := encoding.MarshalAndSaveProtobuf(e.cachePath, e.cache); err != nil {
			e.cacheWriteError = err
		}
		e.cacheLock.Unlock()
	}()

	// Done.
	return result, preservesExecutability, nil, false
}

// stageFromRoot attempts to perform staging from local files by using a reverse
// lookup map.
func (e *endpoint) stageFromRoot(
	path string,
	digest []byte,
	reverseLookupMap *sync.ReverseLookupMap,
	opener *filesystem.Opener,
) bool {
	// See if we can find a path within the root that has a matching digest.
	sourcePath, sourcePathOk := reverseLookupMap.Lookup(digest)
	if !sourcePathOk {
		return false
	}

	// Open the source file and defer its closure.
	source, err := opener.Open(sourcePath)
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

// Stage implements the Stage method for local endpoints.
func (e *endpoint) Stage(paths []string, digests [][]byte) ([]string, []*rsync.Signature, rsync.Receiver, error) {
	// If we're in a read-only mode, we shouldn't be staging files.
	if e.readOnly {
		return nil, nil, nil, errors.New("endpoint is in read-only mode")
	}

	// Verify that we've performed a scan since the last staging operation, that
	// way our count check is valid. If we haven't, then the controller is
	// either malfunctioning or malicious.
	if !e.scannedSinceLastStageCall {
		return nil, nil, nil, errors.New("multiple staging operations performed without scan")
	}
	e.scannedSinceLastStageCall = false

	// Verify that the number of paths provided isn't going to put us over the
	// maximum number of allowed entries.
	if e.maximumEntryCount != 0 && (e.maximumEntryCount-e.lastScanCount) < uint64(len(paths)) {
		return nil, nil, nil, errors.New("staging would exceeded allowed entry count")
	}

	// Generate a reverse lookup map from the cache, which we'll use shortly to
	// detect renames and copies.
	e.cacheLock.Lock()
	reverseLookupMap, err := e.cache.GenerateReverseLookupMap()
	e.cacheLock.Unlock()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to generate reverse lookup map")
	}

	// Create an opener that we can use file opening and defer its closure. We
	// can't cache this across synchronization cycles since its path references
	// may become invalidated or may prevent modifications.
	opener := filesystem.NewOpener(e.root)
	defer opener.Close()

	// Filter the path list by looking for files that we can source locally.
	//
	// First, check if the content can be provided from the stager, which
	// indicates that a previous staging operation was interrupted.
	//
	// Second, use a reverse lookup map (generated from the cache) and see if we
	// can find (and stage) any files locally, which indicates that a file has
	// been copied or renamed.
	//
	// If we manage to handle all files, then we can abort the staging
	// operation.
	filteredPaths := paths[:0]
	for p, path := range paths {
		digest := digests[p]
		if _, err := e.stager.Provide(path, digest); err == nil {
			continue
		} else if e.stageFromRoot(path, digest, reverseLookupMap, opener) {
			continue
		} else {
			filteredPaths = append(filteredPaths, path)
		}
	}
	if len(filteredPaths) == 0 {
		return nil, nil, nil, nil
	}

	// Create an rsync engine.
	engine := rsync.NewEngine()

	// Compute signatures for each of the unstaged paths. For paths that don't
	// exist or that can't be read, just use an empty signature, which means to
	// expect/use an empty base when deltafying/patching.
	signatures := make([]*rsync.Signature, len(filteredPaths))
	for p, path := range filteredPaths {
		if base, err := opener.Open(path); err != nil {
			signatures[p] = &rsync.Signature{}
			continue
		} else if signature, err := engine.Signature(base, 0); err != nil {
			base.Close()
			signatures[p] = &rsync.Signature{}
			continue
		} else {
			base.Close()
			signatures[p] = signature
		}
	}

	// Create a receiver.
	receiver, err := rsync.NewReceiver(e.root, filteredPaths, signatures, e.stager)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to create rsync receiver")
	}

	// Done.
	return filteredPaths, signatures, receiver, nil
}

// Supply implements the supply method for local endpoints.
func (e *endpoint) Supply(paths []string, signatures []*rsync.Signature, receiver rsync.Receiver) error {
	return rsync.Transmit(e.root, paths, signatures, receiver)
}

// Transition implements the Transition method for local endpoints.
func (e *endpoint) Transition(transitions []*sync.Change) ([]*sync.Entry, []*sync.Problem, error) {
	// If we're in a read-only mode, we shouldn't be performing transitions.
	if e.readOnly {
		return nil, nil, errors.New("endpoint is in read-only mode")
	}

	// Verify that we've performed a scan since the last transition operation,
	// that way our count check is valid. If we haven't, then the controller is
	// either malfunctioning or malicious.
	if !e.scannedSinceLastTransitionCall {
		return nil, nil, errors.New("multiple transition operations performed without scan")
	}
	e.scannedSinceLastTransitionCall = false

	// Verify that the number of entries we'll be creating won't put us over the
	// maximum number of allowed entries. Again, we don't worry too much about
	// overflow here for the same reasons as in Entry.Count.
	if e.maximumEntryCount != 0 {
		// Compute the resulting entry count. If we dip below zero in this
		// counting process, then the controller is malfunctioning.
		resultingEntryCount := e.lastScanCount
		for _, transition := range transitions {
			if removed := transition.Old.Count(); removed > resultingEntryCount {
				return nil, nil, errors.New("transition requires removing more entries than exist")
			} else {
				resultingEntryCount -= removed
			}
			resultingEntryCount += transition.New.Count()
		}

		// If the resulting entry count would be too high, then abort the
		// transitioning operation, but return the error as a problem, not an
		// error, since nobody is malfunctioning here.
		results := make([]*sync.Entry, len(transitions))
		for t, transition := range transitions {
			results[t] = transition.Old
		}
		problems := []*sync.Problem{{Error: "transitioning would exceeded allowed entry count"}}
		if e.maximumEntryCount < resultingEntryCount {
			return results, problems, nil
		}
	}

	// Lock and defer release of the cache lock.
	e.cacheLock.Lock()
	defer e.cacheLock.Unlock()

	// Perform the transition.
	results, problems := sync.Transition(
		e.root,
		transitions,
		e.cache,
		e.symlinkMode,
		e.defaultFileMode,
		e.defaultDirectoryMode,
		e.defaultOwnership,
		e.recomposeUnicode,
		e.stager,
	)

	// Wipe the staging directory. We don't monitor for errors here, because we
	// need to return the results and problems no matter what, but if there's
	// something weird going on with the filesystem, we'll see it the next time
	// we scan or stage.
	//
	// TODO: If we see a large number of problems, should we avoid wiping the
	// staging directory? It could be due to an easily correctable error, at
	// which point you wouldn't want to restage if you're talking about lots of
	// files.
	e.stager.wipe()

	// Done.
	return results, problems, nil
}

// Shutdown implements the Shutdown method for local endpoints.
func (e *endpoint) Shutdown() error {
	// Terminate filesystem watching. This will result in the associated events
	// channel being closed.
	e.watchCancel()

	// Done.
	return nil
}
