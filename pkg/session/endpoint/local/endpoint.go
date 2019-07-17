package local

import (
	"context"
	"hash"
	"io"
	"path/filepath"
	syncpkg "sync"
	"time"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/encoding"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/filesystem/behavior"
	"github.com/havoc-io/mutagen/pkg/filesystem/watching"
	"github.com/havoc-io/mutagen/pkg/rsync"
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/sync"
)

const (
	// cacheSaveInterval is the interval at which caches are serialized and
	// written to disk in the background.
	cacheSaveInterval = 60 * time.Second

	// nativeEventsChannelCapacity is the channel to use for events received
	// from native watchers.
	nativeEventsChannelCapacity = 50

	// recheckPathsMaximumCapacity is the maximum capacity for re-check path
	// sets. We also use it as the default map capacity to avoid map
	// reallocation when doing rapid event insertion.
	recheckPathsMaximumCapacity = 50

	// recursiveWatchingEventCoalescingWindow is the time window that recursive
	// watching will wait after an event before strobing the poll events
	// channel. If another event is received during that window, the coalescing
	// timer is reset. It should be small enough to deliver events with no
	// human-perceptible delay, but large enough to group events occurring in
	// rapid succession.
	recursiveWatchingEventCoalescingWindow = 10 * time.Millisecond
)

// endpoint provides a local, in-memory implementation of session.Endpoint for
// local files.
type endpoint struct {
	// root is the synchronization root for the endpoint. This field is static
	// and thus safe for concurrent reads.
	root string
	// readOnly determines whether or not the endpoint should be operating in a
	// read-only mode (i.e. it is the source of unidirectional synchronization).
	// Although the controller can send the endpoint whatever configuration it
	// wants, there may be a configuration validation option for the endpoint
	// that ensures the endpoint is in a read-only state. In those cases, we
	// want to make sure an untrusted controller can't then try to perform a
	// modification operation. This field is static and thus safe for concurrent
	// reads.
	readOnly bool
	// maximumEntryCount is the maximum number of entries within the
	// synchronization root that this endpoint will support synchronizing. This
	// field is static and thus safe for concurrent reads.
	maximumEntryCount uint64
	// probeMode is the probe mode for the session. This field is static and
	// thus safe for concurrent reads.
	probeMode behavior.ProbeMode
	// accelerationAllowed indicates whether or not scan acceleration is allowed
	// for the endpoint. This is computed based off of the scan mode. This field
	// is static and thus safe for concurrent reads.
	accelerationAllowed bool
	// symlinkMode is the symlink mode for the session. This field is static and
	// thus safe for concurrent reads.
	symlinkMode sync.SymlinkMode
	// ignores is the list of ignored paths for the session. This field is
	// static and thus safe for concurrent reads.
	ignores []string
	// defaultFileMode is the default file permission mode to use in "portable"
	// permission propagation. This field is static and thus safe for concurrent
	// reads.
	defaultFileMode filesystem.Mode
	// defaultDirectoryMode is the default directory permission mode to use in
	// "portable" permission propagation. This field is static and thus safe for
	// concurrent reads.
	defaultDirectoryMode filesystem.Mode
	// defaultOwnership is the default ownership specification to use in
	// "portable" permission propagation. This field is static and thus safe for
	// concurrent reads.
	defaultOwnership *filesystem.OwnershipSpecification
	// watchIsRecursive indicates that a watching Goroutine exists and that it
	// is using native recursive watching. This field is static and thus safe
	// for concurrent reads.
	watchIsRecursive bool
	// workerCancel cancels any background worker Goroutines for the endpoint.
	// This field is static and safe for concurrent invocation.
	workerCancel context.CancelFunc
	// pollEvents is the channel used to inform a call to Poll that there are
	// filesystem modifications (and thus it can return). It is a buffered
	// channel with a capacity of one. Senders should always perform a
	// non-blocking send to the channel, because if it is already populated,
	// then filesystem modifications are already indicated. This field is static
	// and never closed, and is thus safe for concurrent send operations.
	pollEvents chan struct{}
	// recursiveWatchRetryEstablish is a channel used by Transition to signal to
	// the recursive watching Goroutine (if any) that it should try to
	// re-establish watching. It is a non-buffered channel, with reads only
	// occurring when the recursive watching Goroutine is waiting to retry watch
	// establishment and writes only occurring in a non-blocking fashion. This
	// field is static and never closed, and is thus safe for concurrent send
	// operations.
	recursiveWatchRetryEstablish chan struct{}
	// recursiveWatchReenableAcceleration is a channel used to signal the
	// recursive watching Goroutine (if any) that acceleration has been disabled
	// (e.g. in Transition) and that it needs to perform a re-scan before
	// re-enabling acceleration. It is a buffered channel with a capacity of
	// one so that notifications can be registered even if the watching
	// Goroutine is handling other events. Senders should always perform a
	// non-blocking send to the channel, because if it is already populated,
	// then the need to re-enable acceleration is already indicated. This field
	// is static and never closed, and is thus safe for concurrent send
	// operations.
	recursiveWatchReenableAcceleration chan struct{}
	// scanLock locks the endpoint's scan-related fields, specifically
	// accelerateScan, snapshot, recheckPaths, hasher, cache, ignoreCache,
	// cacheWriteError, preservesExecutability, decomposesUnicode,
	// lastScanEntryCount, scannedSinceLastStageCall, and
	// scannedSinceLastTransitionCall. This lock is not necessitated by the
	// Endpoint interface (since it doesn't allow concurrent usage), but rather
	// the endpoint's background worker Goroutines for cache saving and
	// (potentially) watching, which also access/modify these fields.
	scanLock syncpkg.Mutex
	// accelerateScan indicates that the Scan function should attempt to
	// accelerate scanning by using data from a background watcher Goroutine.
	accelerateScan bool
	// snapshot is the snapshot from the last scan.
	snapshot *sync.Entry
	// recheckPaths is the set of recheck paths to use when accelerating scans
	// in recursive watching mode. This map will always be initialized (non-nil)
	// and ready for writes.
	recheckPaths map[string]bool
	// hasher is the hasher used for scans.
	hasher hash.Hash
	// cache is the cache from the last successful scan on the endpoint.
	cache *sync.Cache
	// ignoreCache is the ignore cache from the last successful scan on the
	// endpoint.
	ignoreCache sync.IgnoreCache
	// cacheWriteError is the last error encountered when trying to write the
	// cache to disk, if any.
	cacheWriteError error
	// preservesExecutability is the executability preservation behavior
	// detected by the last successful scan on the endpoint.
	preservesExecutability bool
	// decomposesUnicode is the Unicode decomposition behavior detected by the
	// last successful scan on the endpoint.
	decomposesUnicode bool
	// lastScanEntryCount is the entry count at the time of the last scan.
	lastScanEntryCount uint64
	// scannedSinceLastStageCall tracks whether or not a scan operation has
	// occurred since the last staging operation.
	scannedSinceLastStageCall bool
	// scannedSinceLastTransitionCall tracks whether or not a scan operation has
	// occurred since the last transitioning operation.
	scannedSinceLastTransitionCall bool
	// stager is the staging coordinator. It is not safe for concurrent usage,
	// but since Endpoint doesn't allow concurrent usage, we know that the
	// stager will only be used in at most one of Stage or Transition methods at
	// any given time.
	stager *stager
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

	// Determine the maximum entry count.
	maximumEntryCount := configuration.MaximumEntryCount
	if maximumEntryCount == 0 {
		maximumEntryCount = version.DefaultMaximumEntryCount()
	}

	// Determine the maximum staging file size.
	maximumStagingFileSize := configuration.MaximumStagingFileSize
	if maximumStagingFileSize == 0 {
		maximumStagingFileSize = version.DefaultMaximumStagingFileSize()
	}

	// Compute the effective probe mode.
	probeMode := configuration.ProbeMode
	if probeMode.IsDefault() {
		probeMode = version.DefaultProbeMode()
	}

	// Compute the effective scan mode and whether or not scan acceleration is
	// allowed.
	scanMode := configuration.ScanMode
	if scanMode.IsDefault() {
		scanMode = version.DefaultScanMode()
	}
	accelerationAllowed := scanMode == session.ScanMode_ScanModeAccelerated

	// Compute the effective symlink mode.
	symlinkMode := configuration.SymlinkMode
	if symlinkMode.IsDefault() {
		symlinkMode = version.DefaultSymlinkMode()
	}

	// Compute the effective VCS ignore mode.
	ignoreVCSMode := configuration.IgnoreVCSMode
	if ignoreVCSMode.IsDefault() {
		ignoreVCSMode = version.DefaultIgnoreVCSMode()
	}

	// Compute a combined ignore list.
	var ignores []string
	if ignoreVCSMode == sync.IgnoreVCSMode_IgnoreVCSModeIgnore {
		ignores = append(ignores, sync.DefaultVCSIgnores...)
	}
	ignores = append(ignores, configuration.DefaultIgnores...)
	ignores = append(ignores, configuration.Ignores...)

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

	// Compute the cache path.
	var cachePath string
	if endpointOptions.cachePathCallback != nil {
		cachePath, err = endpointOptions.cachePathCallback(sessionIdentifier, alpha)
	} else {
		cachePath, err = pathForCache(sessionIdentifier, alpha)
	}
	if err != nil {
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

	// Compute the effective staging mode.
	stageMode := configuration.StageMode
	if stageMode.IsDefault() {
		stageMode = version.DefaultStageMode()
	}

	// Compute the staging root path and whether or not it should be hidden.
	var stagingRoot string
	var hideStagingRoot bool
	if endpointOptions.stagingRootCallback != nil {
		stagingRoot, hideStagingRoot, err = endpointOptions.stagingRootCallback(sessionIdentifier, alpha)
	} else if stageMode == session.StageMode_StageModeMutagen {
		stagingRoot, err = pathForMutagenStagingRoot(sessionIdentifier, alpha)
	} else if stageMode == session.StageMode_StageModeNeighboring {
		stagingRoot, err = pathForNeighboringStagingRoot(root, sessionIdentifier, alpha)
		hideStagingRoot = true
	} else {
		panic("unhandled staging mode")
	}
	if err != nil {
		return nil, errors.Wrap(err, "unable to compute staging root")
	}

	// Compute the effective watch mode.
	watchMode := configuration.WatchMode
	if watchMode.IsDefault() {
		watchMode = version.DefaultWatchMode()
	}

	// Compute whether or not we're going to use native recursive watching.
	watchIsRecursive := watchMode == session.WatchMode_WatchModePortable &&
		watching.RecursiveWatchingSupported

	// Create a cancellable context in which the endpoint's background worker
	// Goroutines will operate.
	workerContext, workerCancel := context.WithCancel(context.Background())

	// Create the endpoint.
	endpoint := &endpoint{
		root:                               root,
		readOnly:                           readOnly,
		maximumEntryCount:                  maximumEntryCount,
		probeMode:                          probeMode,
		accelerationAllowed:                accelerationAllowed,
		symlinkMode:                        symlinkMode,
		ignores:                            ignores,
		defaultFileMode:                    defaultFileMode,
		defaultDirectoryMode:               defaultDirectoryMode,
		defaultOwnership:                   defaultOwnership,
		watchIsRecursive:                   watchIsRecursive,
		workerCancel:                       workerCancel,
		pollEvents:                         make(chan struct{}, 1),
		recursiveWatchRetryEstablish:       make(chan struct{}),
		recursiveWatchReenableAcceleration: make(chan struct{}, 1),
		recheckPaths:                       make(map[string]bool, recheckPathsMaximumCapacity),
		hasher:                             version.Hasher(),
		cache:                              cache,
		stager: newStager(
			stagingRoot,
			hideStagingRoot,
			version.Hasher(),
			maximumStagingFileSize,
		),
	}

	// Start the cache saving Goroutine.
	go endpoint.saveCacheRegularly(workerContext, cachePath)

	// Compute the effective watch polling interval.
	watchPollingInterval := configuration.WatchPollingInterval
	if watchPollingInterval == 0 {
		watchPollingInterval = version.DefaultWatchPollingInterval()
	}

	// Start the appropriate watching mechanism.
	if watchMode == session.WatchMode_WatchModePortable {
		if watching.RecursiveWatchingSupported {
			go endpoint.watchRecursive(workerContext, watchPollingInterval)
		} else {
			go endpoint.watchPoll(
				workerContext,
				watchPollingInterval,
				watching.NonRecursiveWatchingSupported,
			)
		}
	} else if watchMode == session.WatchMode_WatchModeForcePoll {
		go endpoint.watchPoll(workerContext, watchPollingInterval, false)
	} else if watchMode == session.WatchMode_WatchModeNoWatch {
		// Don't start any watcher.
	} else {
		panic("unhandled watch mode")
	}

	// Success.
	return endpoint, nil
}

// saveCacheRegularly serializes the cache and writes the result to disk at
// regular intervals. It runs as a background Goroutine for all endpoints.
func (e *endpoint) saveCacheRegularly(context context.Context, cachePath string) {
	// Create a ticker to regulate cache saving and defer its shutdown.
	ticker := time.NewTicker(cacheSaveInterval)
	defer ticker.Stop()

	// Track the last saved cache. If it hasn't changed, there's no point in
	// rewriting it. It's safe to keep a reference to the cache since caches are
	// treated as immutable. The only cost is keeping an old cache around until
	// the next write cycle, but that's a relatively small price to pay to avoid
	// unnecessary disk writes.
	var lastSavedCache *sync.Cache

	// Loop indefinitely, watching for cancellation and saving the cache to
	// disk at regular intervals. If we see a cache write failure, we record it,
	// and we don't attempt any more saves. The recorded error will be reported
	// to the controller on the next call to Scan.
	for {
		select {
		case <-context.Done():
			return
		case <-ticker.C:
			e.scanLock.Lock()
			if e.cacheWriteError == nil && e.cache != lastSavedCache {
				if err := encoding.MarshalAndSaveProtobuf(cachePath, e.cache); err != nil {
					e.cacheWriteError = err
				} else {
					lastSavedCache = e.cache
				}
			}
			e.scanLock.Unlock()
		}
	}
}

// stopAndDrainTimer is a convenience function that stops a timer and performs a
// non-blocking drain on its channel. This allows a timer to be stopped/drained
// without any knowledge of its current state.
func stopAndDrainTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

// watchRecursive is the watch loop for platforms where native recursive
// watching facilities are available.
func (e *endpoint) watchRecursive(context context.Context, pollingInterval uint32) {
	// Convert the polling interval to a duration.
	pollingDuration := time.Duration(pollingInterval) * time.Second

	// Create a timer, initially stopped, that we can use to regulate the
	// recreation of watches. Defer its termination in case it's running during
	// cancellation.
	watchRecreationTimer := time.NewTimer(0)
	stopAndDrainTimer(watchRecreationTimer)
	defer watchRecreationTimer.Stop()

	// Create a timer, initially stopped, that we can use to regulate the
	// scanning necessary to establish accelerated watches. Defer its
	// termination in case it's running during cancellation.
	scanTimer := time.NewTimer(0)
	stopAndDrainTimer(scanTimer)
	defer scanTimer.Stop()

	// Create a timer, initially stopped, that we can use to coalesce events.
	// Defer its termination in case it's running during cancellation.
	coalescingTimer := time.NewTimer(0)
	stopAndDrainTimer(coalescingTimer)
	defer coalescingTimer.Stop()

	// Loop and manage watches.
WatchEstablishment:
	for {
		// Create our events channel.
		events := make(chan string, nativeEventsChannelCapacity)

		// Establish a watch and monitor for its termination.
		watchErrors := make(chan error, 1)
		go func() {
			watchErrors <- watching.WatchRecursive(context, e.root, events)
		}()

		// Wait for the watch to be established (as indicated by an empty string
		// on the events channel), or an error to occur. If an error occurs,
		// then we'll strobe poll events and then wait for one polling interval
		// before attempting to re-establish the watch.
		select {
		case <-context.Done():
			return
		case <-watchErrors:
			// If there's a watch error, then something interesting may have
			// happened on disk, so we may as well strobe the poll events
			// channel.
			e.strobePollEvents()

			// Reset the watch recreation timer (which won't be running).
			watchRecreationTimer.Reset(pollingDuration)

			// Wait for cancellation or our next watch creation attempt. We also
			// watch for notifications from Transition, telling us that it might
			// be worth trying to re-establish the watch, because the primary
			// reason for watch errors here is non-existence of the
			// synchronization root.
			select {
			case <-context.Done():
				return
			case <-watchRecreationTimer.C:
				continue WatchEstablishment
			case <-e.recursiveWatchRetryEstablish:
				stopAndDrainTimer(watchRecreationTimer)
				continue WatchEstablishment
			}
		case path := <-events:
			if path != "" {
				panic("watch initialization path non-empty")
			}
		}

		// Now that the watch has been successfully established, strobe the poll
		// events channel. The reason for this is that, for recursive watching,
		// successful establishment of the watch serves as a signal that the
		// synchronization root exists and is accessible, which may not have
		// been the case previously. For example, if the root or a parent didn't
		// exist, we would have looped while trying to establish a watch,
		// strobing the poll events channel on each failure. If we suddenly
		// succeed, we need to notify the controller, otherwise it won't see any
		// events until we receive events from within the root. Essentially,
		// successful establishment of the watch is an event, of sorts, that
		// we're only in a position to report here.
		e.strobePollEvents()

		// Reset the scan timer (which won't be running) to fire immediately in
		// our watch loop.
		scanTimer.Reset(0)

		// Loop and process events.
	EventProcessing:
		for {
			select {
			case <-context.Done():
				return
			case <-scanTimer.C:
				// If acceleration isn't allowed on the endpoint, then we don't
				// need to perform a baseline scan, so we can just continue
				// watching.
				if !e.accelerationAllowed {
					continue EventProcessing
				}

				// Attempt to perform a full (warm) baseline scan. If this
				// succeeds, then we can enable accelerated scanning (since our
				// watch is established and we'll see all events after the
				// baseline scan time). If this fails, then we'll reset the scan
				// timer (which won't be running) and try again after the
				// polling interval.
				e.scanLock.Lock()
				if err := e.scan(nil, nil); err != nil {
					scanTimer.Reset(pollingDuration)
				} else {
					e.accelerateScan = true
				}
				e.scanLock.Unlock()
			case <-e.recursiveWatchReenableAcceleration:
				// If acceleration isn't allowed on the endpoint, then we don't
				// need to re-enable it, so we can just continue watching.
				if !e.accelerationAllowed {
					continue EventProcessing
				}

				// It's possible that the scan timer is running (e.g. if we
				// receive this signal before we finish the initial
				// acceleration-enabling scan), so we ensure it's stopped (since
				// the following scan will serve the same purpose).
				stopAndDrainTimer(scanTimer)

				// Attempt to perform a full (warm) baseline scan. If this
				// succeeds, then we can re-enable acceleration. If this fails,
				// then we'll reset the scan timer (which won't be running) and
				// try again after the polling interval.
				e.scanLock.Lock()
				if err := e.scan(nil, nil); err != nil {
					scanTimer.Reset(pollingDuration)
				} else {
					e.accelerateScan = true
				}
				e.scanLock.Unlock()
			case <-watchErrors:
				// If acceleration is allowed on the endpoint, then disable scan
				// acceleration and clear out the re-check path set.
				if e.accelerationAllowed {
					e.scanLock.Lock()
					e.accelerateScan = false
					e.recheckPaths = make(map[string]bool, recheckPathsMaximumCapacity)
					e.scanLock.Unlock()
				}

				// Stop and drain any timers that might be running.
				stopAndDrainTimer(scanTimer)
				stopAndDrainTimer(coalescingTimer)

				// Strobe the poll events channel. This is necessary since there
				// may have been a scan performed with stale re-check paths in
				// the window between the watch failure and the time that we
				// were able to disable acceleration. Note that we don't have to
				// worry about a (partially or completely) pre-Transition
				// snapshot being returned in this window (and driving a
				// feedback loop) because Transition operations always disable
				// scan acceleration (see the comment in Transition). Worst case
				// scenario we'd have returned a slightly stale scan. Even if
				// the endpoint doesn't allow acceleration, the failure of the
				// watch is still worth reporting.
				e.strobePollEvents()

				// Reset the watch recreation timer (which won't be running).
				watchRecreationTimer.Reset(pollingDuration)

				// Wait for cancellation or watch recreation. Note that, unlike
				// above, we don't watch for notifications from Transition here
				// because watch errors here are likely due to event overflow
				// and we're better off giving the filesystem some time to
				// settle.
				select {
				case <-context.Done():
					return
				case <-watchRecreationTimer.C:
					continue WatchEstablishment
				}
			case path := <-events:
				// If the path is a temporary file generated by Mutagen, then
				// ignore it. We can use our fast-path base computation since
				// recursive watchers generate synchronization-root-relative
				// paths.
				if filesystem.IsTemporaryFileName(sync.PathBase(path)) {
					continue EventProcessing
				}

				// If acceleration is allowed on the endpoint, then register the
				// event path as a re-check path. If the re-check paths set
				// would overflow its allowed size, then temporarily disable
				// acceleration, clear out the re-check path set, and reset the
				// scan timer (which may or may not be running) to force a full
				// (warm) scan (and re-enable acceleration).
				if e.accelerationAllowed {
					e.scanLock.Lock()
					if len(e.recheckPaths) == recheckPathsMaximumCapacity {
						e.accelerateScan = false
						e.recheckPaths = make(map[string]bool, recheckPathsMaximumCapacity)
						stopAndDrainTimer(scanTimer)
						scanTimer.Reset(pollingDuration)
					}
					e.recheckPaths[path] = true
					e.scanLock.Unlock()
				}

				// Reset the coalescing timer (which may or may not be running).
				stopAndDrainTimer(coalescingTimer)
				coalescingTimer.Reset(recursiveWatchingEventCoalescingWindow)
			case <-coalescingTimer.C:
				// Strobe the poll events channel.
				e.strobePollEvents()
			}
		}
	}
}

// watchPoll is the watch loop for poll-based watching, with optional support
// for using native non-recursive watching facilities to reduce notification
// latency on frequently updated contents.
func (e *endpoint) watchPoll(
	context context.Context,
	pollingInterval uint32,
	useNonRecursiveWatching bool,
) {
	// Create a ticker to regulate polling and defer its shutdown.
	ticker := time.NewTicker(time.Duration(pollingInterval) * time.Second)
	defer ticker.Stop()

	// Track whether or not it's our first iteration in the polling loop. We
	// adjust some behaviors in that case.
	first := true

	// Track the previous scan results that we want to compare to watch for
	// changes. It's safe to keep a reference to the snapshot since entries are
	// treated as immutable.
	var previousSnapshot *sync.Entry
	var previousPreservesExecutability, previousDecomposesUnicode bool

	// If non-recursive watching is enabled, attempt to set up the non-recursive
	// watching infrastructure.
	var nonRecursiveWatchEvents chan string
	var nonRecursiveWatcher *watching.NonRecursiveMRUWatcher
	var nonRecursiveWatcherErrors chan error
	var coalescingTimer *time.Timer
	var coalescingTimerEvents <-chan time.Time
	if useNonRecursiveWatching {
		// Set up the events channel.
		nonRecursiveWatchEvents = make(chan string, nativeEventsChannelCapacity)

		// Attempt to create the watcher. If this fails, we simply avoid using
		// native watching.
		var err error
		nonRecursiveWatcher, err = watching.NewNonRecursiveMRUWatcher(nonRecursiveWatchEvents, 0)
		if err == nil {
			// Extract the watcher's errors channel.
			nonRecursiveWatcherErrors = nonRecursiveWatcher.Errors

			// Set up conditional shutdown of the watcher. If it dies before
			// watching terminates, we'll nil it out.
			defer func() {
				if nonRecursiveWatcher != nil {
					nonRecursiveWatcher.Stop()
				}
			}()

			// Create a timer, initially stopped, that we can use to coalesce
			// events. Defer its termination in case it's running during
			// cancellation.
			coalescingTimer = time.NewTimer(0)
			stopAndDrainTimer(coalescingTimer)
			defer coalescingTimer.Stop()

			// Extract the timer's event channel.
			coalescingTimerEvents = coalescingTimer.C
		}
	}

	// Loop until cancellation, performing polling at the specified interval.
	for {
		// Set behaviors based on whether or not this is our first time in the
		// loop. If this is our first time in the loop, then we skip waiting,
		// because our ticker won't fire its first event until after the polling
		// duration has elapsed, and we'd like a baseline scan before that. The
		// reason we want a baseline scan before that is that we'll ignore
		// modifications on our first successful scan. The reason for ignoring
		// these modifications is that we'll be comparing against zero-valued
		// variables and are thus certain to see modifications if there is
		// existing content on disk. Since the controller already skips polling
		// (if watching is enabled) on its first synchronization cycle, there's
		// no point for us to also send a notification, because if both
		// endpoints did this, you'd see up to three scans on session startup.
		// Of course, if our scan fails on the first try, then we'll allow a
		// notification (due to these "artificial" modifications) to be sent
		// after the first successful scan, but that will at least occur after
		// the initial polling duration.
		var skipWaiting, ignoreModifications bool
		if first {
			skipWaiting = true
			ignoreModifications = true
			first = false
		}

		// Unless we're skipping waiting, wait for cancellation, a tick event,
		// a notification from our non-recursive watches, or a coalesced event.
		if !skipWaiting {
			select {
			case <-context.Done():
				return
			case <-ticker.C:
			case <-nonRecursiveWatcherErrors:
				// Terminate the watcher and nil it out. We don't bother trying
				// to re-establish it. Also nil out the errors channel in case
				// the watcher pumps any additional errors into it (in which
				// case we don't want to trigger this code again on a nil
				// watcher). We'll allow event channels to continue since they
				// may contain residual events.
				nonRecursiveWatcher.Stop()
				nonRecursiveWatcher = nil
				nonRecursiveWatcherErrors = nil
				continue
			case path := <-nonRecursiveWatchEvents:
				// If the path is a temporary file generated by Mutagen, then
				// ignore it. Unlike in the recurisve watcher case, we can't use
				// our fast-path base computation since non-recursive watching
				// won't return root-relative paths.
				if filesystem.IsTemporaryFileName(filepath.Base(path)) {
					continue
				}

				// Reset the coalescing timer (which may or may not be running)
				// and continue. Once it fires, we'll perform a rescan.
				stopAndDrainTimer(coalescingTimer)
				coalescingTimer.Reset(recursiveWatchingEventCoalescingWindow)
				continue
			case <-coalescingTimerEvents:
			}
		}

		// Grab the scan lock.
		e.scanLock.Lock()

		// Disable the use of the existing scan results.
		e.accelerateScan = false

		// Perform a scan. If there's an error, then assume it's due to
		// concurrent modification. In that case, release the scan lock and
		// strobe the poll events channel. The controller can then perform a
		// full scan.
		if err := e.scan(nil, nil); err != nil {
			e.scanLock.Unlock()
			e.strobePollEvents()
			continue
		}

		// If our scan was successful, then we know that the scan results
		// will be okay to return for the next Scan call, though we only
		// indicate that acceleration should be used if the endpoint allows it.
		e.accelerateScan = e.accelerationAllowed

		// Extract scan parameters so that we can release the scan lock.
		snapshot := e.snapshot
		preservesExecutability := e.preservesExecutability
		decomposesUnicode := e.decomposesUnicode

		// Release the scan lock.
		e.scanLock.Unlock()

		// Check for modifications.
		modified := !snapshot.Equal(previousSnapshot) ||
			preservesExecutability != previousPreservesExecutability ||
			decomposesUnicode != previousDecomposesUnicode

		// If we have a working non-recursive watcher, then perform a full diff
		// to determine new watch paths, and then start the new watches. Any
		// watch errors will be reported on the watch errors channel.
		if nonRecursiveWatcher != nil {
			changes := sync.Diff(previousSnapshot, snapshot)
			for _, change := range changes {
				nonRecursiveWatcher.Watch(filepath.Join(e.root, change.Path))
			}
		}

		// Update our tracking parameters.
		previousSnapshot = snapshot
		previousPreservesExecutability = preservesExecutability
		previousDecomposesUnicode = decomposesUnicode

		// If we've seen modifications, and we're not ignoring them, then strobe
		// the poll events channel.
		if modified && !ignoreModifications {
			e.strobePollEvents()
		}
	}
}

// strobePollEvents is a convenience function to strobe the pollEvents channel
// in a non-blocking fashion.
func (e *endpoint) strobePollEvents() {
	select {
	case e.pollEvents <- struct{}{}:
	default:
	}
}

// Poll implements the Poll method for local endpoints.
func (e *endpoint) Poll(context context.Context) error {
	// Wait for either cancellation or an event.
	select {
	case <-context.Done():
	case _, ok := <-e.pollEvents:
		if !ok {
			panic("poll events channel closed")
		}
	}

	// Done.
	return nil
}

// scan is the internal function which performs a scan operation on the root and
// updates the endpoint scan parameters. The caller must hold the endpoint's
// scan lock.
func (e *endpoint) scan(baseline *sync.Entry, recheckPaths map[string]bool) error {
	// Perform a full (warm) scan, watching for errors.
	snapshot, preservesExecutability, decomposesUnicode, newCache, newIgnoreCache, err := sync.Scan(
		e.root,
		baseline, recheckPaths,
		e.hasher, e.cache,
		e.ignores, e.ignoreCache,
		e.probeMode,
		e.symlinkMode,
	)
	if err != nil {
		return err
	}

	// Update the internal snapshot.
	e.snapshot = snapshot

	// Update caches.
	e.cache = newCache
	e.ignoreCache = newIgnoreCache

	// Update behavior data.
	e.preservesExecutability = preservesExecutability
	e.decomposesUnicode = decomposesUnicode

	// Update the last scan entry count.
	e.lastScanEntryCount = snapshot.Count()

	// Update call states.
	e.scannedSinceLastStageCall = true
	e.scannedSinceLastTransitionCall = true

	// Success.
	return nil
}

// Scan implements the Scan method for local endpoints.
func (e *endpoint) Scan(_ *sync.Entry, full bool) (*sync.Entry, bool, error, bool) {
	// Grab the scan lock and defer its release.
	e.scanLock.Lock()
	defer e.scanLock.Unlock()

	// Before attempting to perform a scan, check for any cache write errors
	// that may have occurred during background cache writes. If we see any
	// error, then we skip scanning and report them here.
	if e.cacheWriteError != nil {
		return nil, false, errors.Wrap(e.cacheWriteError, "unable to save cache to disk"), false
	}

	// Perform a scan.
	//
	// We check to see if we can accelerate the scanning process by using
	// information from a background watching Goroutine. For recursive watching,
	// this means performing a re-scan using a baseline and a set of re-check
	// paths. For poll-based watching, this just means re-using the last scan,
	// so no action is needed here. If acceleration isn't available (due to the
	// state of the watcher or because it's disallowed on the endpoint), then we
	// just perform a full (warm) scan. We also avoid acceleration in the event
	// that a full scan has been explicitly requested, but we don't make any
	// change to the state of acceleration availability, because performing a
	// full warm scan will only improve the accuracy of the baseline (most
	// recent) snapshot, so acceleration will still work.
	//
	// If we see any error while scanning, we just have to assume that it's due
	// to concurrent modifications and suggest a retry.
	if e.accelerateScan && !full {
		if e.watchIsRecursive {
			if err := e.scan(e.snapshot, e.recheckPaths); err != nil {
				return nil, false, err, true
			} else {
				e.recheckPaths = make(map[string]bool, recheckPathsMaximumCapacity)
			}
		}
	} else {
		if err := e.scan(nil, nil); err != nil {
			return nil, false, err, true
		}
	}

	// Verify that we haven't exceeded the maximum entry count.
	if e.lastScanEntryCount > e.maximumEntryCount {
		return nil, false, errors.New("exceeded allowed entry count"), true
	}

	// Success.
	return e.snapshot, e.preservesExecutability, nil, false
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

	// Grab the scan lock. We'll need this to verify the last scan entry count
	// and to generate the reverse lookup map.
	e.scanLock.Lock()

	// Verify that we've performed a scan since the last staging operation, that
	// way our count check is valid. If we haven't, then the controller is
	// either malfunctioning or malicious.
	if !e.scannedSinceLastStageCall {
		e.scanLock.Unlock()
		return nil, nil, nil, errors.New("multiple staging operations performed without scan")
	}
	e.scannedSinceLastStageCall = false

	// Verify that the number of paths provided isn't going to put us over the
	// maximum number of allowed entries.
	if e.maximumEntryCount != 0 && (e.maximumEntryCount-e.lastScanEntryCount) < uint64(len(paths)) {
		e.scanLock.Unlock()
		return nil, nil, nil, errors.New("staging would exceeded allowed entry count")
	}

	// Generate a reverse lookup map from the cache, which we'll use shortly to
	// detect renames and copies.
	reverseLookupMap, err := e.cache.GenerateReverseLookupMap()
	if err != nil {
		e.scanLock.Unlock()
		return nil, nil, nil, errors.Wrap(err, "unable to generate reverse lookup map")
	}

	// Release the scan lock.
	e.scanLock.Unlock()

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
func (e *endpoint) Transition(transitions []*sync.Change) ([]*sync.Entry, []*sync.Problem, bool, error) {
	// If we're in a read-only mode, we shouldn't be performing transitions.
	if e.readOnly {
		return nil, nil, false, errors.New("endpoint is in read-only mode")
	}

	// Grab the scan lock and defer its release.
	e.scanLock.Lock()
	defer e.scanLock.Unlock()

	// Verify that we've performed a scan since the last transition operation,
	// that way our count check is valid. If we haven't, then the controller is
	// either malfunctioning or malicious.
	if !e.scannedSinceLastTransitionCall {
		return nil, nil, false, errors.New("multiple transition operations performed without scan")
	}
	e.scannedSinceLastTransitionCall = false

	// Verify that the number of entries we'll be creating won't put us over the
	// maximum number of allowed entries. Again, we don't worry too much about
	// overflow here for the same reasons as in Entry.Count.
	if e.maximumEntryCount != 0 {
		// Compute the resulting entry count. If we dip below zero in this
		// counting process, then the controller is malfunctioning.
		resultingEntryCount := e.lastScanEntryCount
		for _, transition := range transitions {
			if removed := transition.Old.Count(); removed > resultingEntryCount {
				return nil, nil, false, errors.New("transition requires removing more entries than exist")
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
			return results, problems, false, nil
		}
	}

	// Perform the transition.
	results, problems, stagerMissingFiles := sync.Transition(
		e.root,
		transitions,
		e.cache,
		e.symlinkMode,
		e.defaultFileMode,
		e.defaultDirectoryMode,
		e.defaultOwnership,
		e.decomposesUnicode,
		e.stager,
	)

	// In case there's a recursive watching Goroutine that doesn't currently
	// have a watch established (due to non-existence of the synchronization
	// root), send a signal that watch establishment should be retried
	// immediately, because Transition likely created the synchronization root
	// in that case. If the Goroutine already has a watch established, then this
	// is a no-op.
	if e.watchIsRecursive {
		select {
		case e.recursiveWatchRetryEstablish <- struct{}{}:
		default:
		}
	}

	// Disable scan acceleration. It's critical to do this after a Transition
	// operation to avoid Scan returning a pre-Transition snapshot. This can
	// happen in poll-based watching if the last snapshot is pre-Transition and
	// in recursive watching if the Transition errors-out the watch (causing
	// events to be lost and a (partially or completely) pre-Transition
	// baseline-based scan to be generated in the small window before the watch
	// error is detected). Both of these "failure" modes can happen in normal
	// operation (with externally generated events), but it is problematic in
	// the case of Transition because returning a pre-Transition snapshot can
	// lead to a feedback loop between endpoints (depending on the phase and
	// mode of their watching Goroutines) where they essentially swap returned
	// snapshots each time (one pre-Transition and one post-Transition) and
	// changes are continually inverted and bounced back and forth between the
	// endpoints. This isn't just hypothetical - it's quite easy to reproduce
	// with poll-based watching and a large(ish) polling interval (e.g. a few
	// seconds). The reason we don't need to worry about scans being stale in
	// normal operation is that there isn't a corresponding actor on the other
	// endpoint that's returning snapshots with constantly inverted changes
	// (that effectively invert the algebra of the reconciliation algorithm on
	// each synchronization cycle), driving a feedback loop. For poll-based
	// scanning, acceleration will be re-enabled the next time it performs a
	// scan, while for recursive watching we need to manually signal that it
	// should attempt to re-enable accelated scanning.
	e.accelerateScan = false
	if e.watchIsRecursive {
		select {
		case e.recursiveWatchReenableAcceleration <- struct{}{}:
		default:
		}
	}

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
	return results, problems, stagerMissingFiles, nil
}

// Shutdown implements the Shutdown method for local endpoints.
func (e *endpoint) Shutdown() error {
	// Mark background worker Goroutines for termination. We don't wait for
	// their termination since it will be almost instant and there's not any
	// important reason to synchronize their shutdown. The worst case scenario
	// resulting from their continued execution after a return from this
	// function would be one cache write occurring after the creation of a new
	// endpoint using the same cache path, but this is extremely unlikely and
	// not problematic if it does occur.
	e.workerCancel()

	// Done.
	return nil
}
