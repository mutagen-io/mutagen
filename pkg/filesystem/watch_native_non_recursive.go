// +build linux

package filesystem

import (
	contextpkg "context"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/golang/groupcache/lru"
)

const (
	// watchNativeNonRecursiveMaximumWatches is the maximum number of watches
	// allowed on systems that require a watch/file descriptor for each watch
	// and don't support recursive watching.
	watchNativeNonRecursiveMaximumWatches = 50
)

// watchNative attempts to perform efficient watching using the operating
// system's native filesystem watching facilities.
func watchNative(context contextpkg.Context, root string, events chan struct{}, pollInterval uint32) error {
	// Validate the polling interval and convert it to a duration.
	if pollInterval == 0 {
		return errors.New("polling interval must be greater than 0 seconds")
	}
	pollIntervalDuration := time.Duration(pollInterval) * time.Second

	// Create a timer to regulate polling. Start it with a 0 duration so that
	// the first polling takes place immediately. Subsequent pollings will take
	// place at the normal interval.
	pollingTimer := time.NewTimer(0)

	// Create a timer that we can use to coalesce events. It will be created
	// running, so make sure to stop it and consume its first event, if any.
	coalescingTimer := time.NewTimer(watchNativeCoalescingWindow)
	if !coalescingTimer.Stop() {
		<-coalescingTimer.C
	}

	// Compute the path to the root's parent directory and the root leaf name.
	rootParentPath, rootLeafName := filepath.Split(root)

	// Create a watcher for the root path (by watching its parent directory) and
	// defer its shutdown.
	rootParentWatcher, err := newNonRecursiveWatcher()
	if err != nil {
		return errors.Wrap(err, "unable to create root parent watcher")
	}
	defer rootParentWatcher.stop()

	// Create parameters to track the state of the root parent directory.
	var rootParentExists bool
	var rootParentMetadata os.FileInfo
	var rootParentWatched bool

	// Create a watcher for paths at or beneath the root and defer its shutdown.
	watcher, err := newNonRecursiveWatcher()
	if err != nil {
		return errors.Wrap(err, "unable to create watcher")
	}
	defer watcher.stop()

	// Create a map to track the paths that are currently watched, along with
	// their associated metadata.
	watchedPaths := make(map[string]os.FileInfo, watchNativeNonRecursiveMaximumWatches-1)

	// Create an LRU-evicting cache to manage eviction of watched paths. Its
	// size is one less than the maximum value since we have to account for the
	// root parent watch.
	//
	// The keys in this cache are paths and the values are simply 0-valued
	// integers. We only use it to track path watching order and to manage the
	// eviction process. There's no need to defer the eviction of all entries -
	// we simply rely on the watcher to cancel all the watches once this
	// function returns.
	//
	// Because the eviction API doesn't allow us to return an error on unwatch
	// failure, we create a channel that the polling loop can monitor for
	// failure.
	unwatchErrors := make(chan error, 1)
	watchedPathManager := lru.New(watchNativeNonRecursiveMaximumWatches - 1)
	watchedPathManager.OnEvicted = func(key lru.Key, _ interface{}) {
		if path, ok := key.(string); !ok {
			panic("invalid key type in watch path cache")
		} else {
			if err := watcher.unwatch(path); err != nil {
				select {
				case unwatchErrors <- err:
				default:
				}
			}
			delete(watchedPaths, path)
		}
	}

	// Start a cancellable Goroutine to extract events/errors from the watchers
	// and manage the coalescing timer. Defer cancellation of this Goroutine and
	// monitor for its failure.
	monitoringContext, monitoringCancel := contextpkg.WithCancel(contextpkg.Background())
	defer monitoringCancel()
	monitoringErrors := make(chan error, 1)
	go func() {
		for {
			// Wait for an event or error.
			var resetCoalescingTimer bool
			select {
			case <-monitoringContext.Done():
				monitoringErrors <- errors.New("monitoring cancelled")
				return
			case path, ok := <-rootParentWatcher.eventPaths:
				if !ok {
					monitoringErrors <- errors.New("root parent watcher event stream closed")
					return
				} else if filepath.Base(path) == rootLeafName {
					resetCoalescingTimer = true
				}
			case path, ok := <-watcher.eventPaths:
				if !ok {
					monitoringErrors <- errors.New("watcher event stream closed")
					return
				}
				resetCoalescingTimer = !IsTemporaryFileName(filepath.Base(path))
			}

			// Reset the coalescing timer if necessary. Perform a non-blocking
			// drain since we don't know if the timer was running or not.
			if resetCoalescingTimer {
				if !coalescingTimer.Stop() {
					select {
					case <-coalescingTimer.C:
					default:
					}
				}
				coalescingTimer.Reset(watchNativeCoalescingWindow)
			}
		}
	}()

	// Create a container to track polling contents.
	var contents map[string]os.FileInfo

	// Poll for cancellation, unwatching failures, the coalescing timer, or the
	// polling timer.
	for {
		select {
		case <-context.Done():
			// Abort the watch.
			return errors.New("watch cancelled")
		case err := <-unwatchErrors:
			// Abort the watch.
			return errors.Wrap(err, "unable to unwatch path")
		case <-coalescingTimer.C:
			// Forward a coalesced event in a non-blocking fashion.
			select {
			case events <- struct{}{}:
			default:
			}
		case <-pollingTimer.C:
			// Perform a scan. If there's an error, then reset the timer and try
			// again. We have to assume that errors here are due to concurrent
			// modifications, so there's not much we can do to handle them.
			// Concurrent modifications will also put a stop to any other lstat
			// operations we want to try, so there's no point in still trying
			// those.
			newContents, changed, changes, err := poll(root, contents, true)
			if err != nil {
				pollingTimer.Reset(pollIntervalDuration)
				continue
			}

			// Store the new contents.
			contents = newContents

			// If there's been a change, then send a notification in a
			// non-blocking fashion.
			if changed {
				select {
				case events <- struct{}{}:
				default:
				}
			}

			// Grab current root parent parameters.
			var rootParentCurrentlyExists bool
			var currentRootParentMetadata os.FileInfo
			if m, err := os.Lstat(rootParentPath); err != nil {
				if !os.IsNotExist(err) {
					return errors.Wrap(err, "unable to probe root parent metadata")
				}
			} else {
				rootParentCurrentlyExists = true
				currentRootParentMetadata = m
			}

			// Check if we need to re-establish the root parent watch.
			reestablishRootParentWatch := rootParentCurrentlyExists != rootParentExists ||
				!watchRootParametersEqual(currentRootParentMetadata, rootParentMetadata)

			// Re-establish the root parent watch if necessary.
			if reestablishRootParentWatch {
				// Remove any existing watch.
				if rootParentWatched {
					if err := rootParentWatcher.unwatch(rootParentPath); err != nil {
						return errors.Wrap(err, "unable to remove stale root parent watch")
					}
					rootParentWatched = false
				}

				// If the root parent currently exists, then attempt to start
				// watching.
				if rootParentCurrentlyExists {
					if err := rootParentWatcher.watch(rootParentPath); err != nil {
						if os.IsNotExist(err) {
							rootParentCurrentlyExists = false
							currentRootParentMetadata = nil
						} else {
							return errors.Wrap(err, "unable to watch root parent path")
						}
					} else {
						rootParentWatched = true
					}
				}

				// Unlike the recursive case, we don't send a notification here
				// because the poll check will have seen any changes and
				// reported them above.
			}

			// Update root parent parameters.
			rootParentExists = rootParentCurrentlyExists
			rootParentMetadata = currentRootParentMetadata

			// Remove existing watches for paths that have been deleted or
			// modified (if modified, we'll re-add them below).
			for p, m := range watchedPaths {
				currentMetadata, ok := newContents[p]
				if ok && watchRootParametersEqual(currentMetadata, m) {
					continue
				}
				watchedPathManager.Remove(p)
			}

			// Filter any potential watch paths that are already watched.
			for p := range changes {
				if _, ok := watchedPaths[p]; ok {
					delete(changes, p)
				}
			}

			// If the new changes are too numerous to watch on their own, then
			// just ignore them. This generally happens on massive bulk creates,
			// where we wouldn't want to watch all the new files anyway. If the
			// paths were in depth-first-traversal-ordering, then it might make
			// sense (as an heuristic) to take the first chunk of them, but they
			// aren't, and it's a questionable heuristic anyway.
			if len(changes) > (watchNativeNonRecursiveMaximumWatches - 1) {
				changes = nil
			}

			// Add new watches.
			for p := range changes {
				if err := watcher.watch(p); err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return errors.Wrap(err, "unable to create watch")
				}
				watchedPaths[p] = newContents[p]
				watchedPathManager.Add(p, 0)
			}

			// Reset the polling timer and continue polling.
			pollingTimer.Reset(pollIntervalDuration)
		}
	}
}
