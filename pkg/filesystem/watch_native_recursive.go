// +build windows darwin,cgo

package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pkg/errors"
)

const (
	watchRootParameterPollingInterval = 1 * time.Second
	watchRestartWait                  = 1 * time.Second
)

// isParentOrSelf returns true if and only if parent is a parent path of child
// or equal to child. It is designed to be cheap at the (potential) cost of
// correctness, but it is only designed for internal usage with file
// notifications, so this is probably acceptable. It assumes UTF-8 encoding.
func isParentOrSelf(parent, child string) bool {
	parentLength := len(parent)
	childLength := len(child)
	if childLength < parentLength {
		return false
	} else if parent != child[:parentLength] {
		return false
	} else if childLength > parentLength {
		return os.IsPathSeparator(child[parentLength])
	}
	return true
}

// watchNative attempts to perform efficient watching using the operating
// system's native filesystem watching facilities.
func watchNative(context context.Context, root string, events chan struct{}, _ uint32) error {
	// Compute the watch root. If we're on macOS, this will be the root itself.
	// If we're on Windows, this will be the parent directory of the root.
	var watchRoot string
	if runtime.GOOS == "darwin" {
		watchRoot = root
	} else if runtime.GOOS == "windows" {
		watchRoot = filepath.Dir(root)
	} else {
		panic("unhandled platform case")
	}

	// Set up initial watch root parameters.
	var watchRootExists bool
	var watchRootMetadata os.FileInfo
	var forceRecreate bool

	// Set up our initial event paths channel.
	dummyEventPaths := make(chan string)
	eventPaths := dummyEventPaths

	// Create a placeholder for the watch.
	var watch *recursiveWatch

	// Create a timer that we can use to coalesce events. It will be created
	// running, so make sure to stop it and consume its first event, if any.
	coalescingTimer := time.NewTimer(watchNativeCoalescingWindow)
	if !coalescingTimer.Stop() {
		<-coalescingTimer.C
	}

	// Create a timer to watch for changes to the watch root device ID and/or
	// inode. Start it with a 0 duration so that the first check takes place
	// immediately. Subsequent checks will take place at the normal interval.
	watchRootCheckTimer := time.NewTimer(0)

	// Create a function to clean up after ourselves.
	defer func() {
		// Cancel any watch.
		if watch != nil {
			watch.stop()
		}

		// Cancel the coalescing timer.
		coalescingTimer.Stop()

		// Cancel the root check timer.
		watchRootCheckTimer.Stop()
	}()

	// Poll for cancellation, the next raw event, the coalescing timer, or the
	// root check timer.
	for {
		select {
		case <-context.Done():
			// Abort the watch.
			return errors.New("watch cancelled")
		case path, ok := <-eventPaths:
			// If the event channel has been closed, then something's gone wrong
			// with the watch (e.g. a buffer overflow in ReadDirectoryChangesW),
			// but it should be a recoverable error, so we need to recreate the
			// watch.
			if !ok {
				// Close out the watch and clear the event channel. We know that
				// the watch is non-nil here because it's set at the same time
				// as eventPaths, from which we just read (the dummy event
				// channel never closes).
				watch.stop()
				watch = nil
				eventPaths = dummyEventPaths

				// Mark forced recreation.
				forceRecreate = true

				// Trigger the root check timer to re-run after a short delay.
				// We could set it to run immediately, but if the error is due
				// to rapid disk events causing a read buffer overflow, then
				// we're better off just waiting until that's done, otherwise
				// we'll just burn CPU cycles recreating over and over again.
				if !watchRootCheckTimer.Stop() {
					<-watchRootCheckTimer.C
				}
				watchRootCheckTimer.Reset(watchRestartWait)

				// Continue polling.
				continue
			}

			// Handle the path appropriately.
			// NOTE: When using FSEvents, event paths are (a) relative to the
			// device root and (b) fully resolved in terms of symlinks. This
			// means that isParentOrSelf will not work. Fortunately,
			// isParentOrSelf isn't necessary when using FSEvents since we watch
			// the root itself.
			if IsTemporaryFileName(filepath.Base(path)) {
				// Ignore intermediate temporary files.
				continue
			} else if runtime.GOOS == "windows" && !isParentOrSelf(root, path) {
				// If we're on Windows, then we're monitoring the parent
				// directory of the synchronization root, so if the notification
				// is for a path outside that root, ignore it.
				continue
			} else {
				// Otherwise we're looking at a relevant event, so reset the
				// coalescing timer. Perform a non-blocking drain since we don't
				// know if the timer was running or not.
				if !coalescingTimer.Stop() {
					select {
					case <-coalescingTimer.C:
					default:
					}
				}
				coalescingTimer.Reset(watchNativeCoalescingWindow)
			}
		case <-coalescingTimer.C:
			// Forward a coalesced event in a non-blocking fashion.
			select {
			case events <- struct{}{}:
			default:
			}
		case <-watchRootCheckTimer.C:
			// Grab current watch root parameters.
			var watchRootCurrentlyExists bool
			var currentWatchRootMetadata os.FileInfo
			if m, err := os.Lstat(watchRoot); err != nil {
				if !os.IsNotExist(err) {
					return errors.Wrap(err, "unable to probe root metadata")
				}
			} else {
				watchRootCurrentlyExists = true
				currentWatchRootMetadata = m
			}

			// Check if we need to recreate the watcher.
			recreate := forceRecreate ||
				watchRootCurrentlyExists != watchRootExists ||
				!watchRootParametersEqual(currentWatchRootMetadata, watchRootMetadata)

			// HACK: On macOS, it's not necessary to recreate a watcher for a
			// file root, even if the file itself has changed.
			if recreate && runtime.GOOS == "darwin" {
				overrideRecreate := !forceRecreate &&
					watchRootExists && watchRootCurrentlyExists &&
					currentWatchRootMetadata.Mode() == watchRootMetadata.Mode() &&
					currentWatchRootMetadata.Mode()&os.ModeType == 0
				if overrideRecreate {
					recreate = false
				}
			}

			// Unmark forced recreation.
			forceRecreate = false

			// Recreate the watcher if necessary.
			if recreate {
				// Close out any existing watcher and reset the event paths
				// channel.
				if watch != nil {
					watch.stop()
					watch = nil
					eventPaths = dummyEventPaths
				}

				// If the watch root exists, then attempt to start watching. If
				// the restart attempt fails, then just mark a forced
				// recreation and try again after a delay.
				if watchRootCurrentlyExists {
					if w, err := newRecursiveWatch(watchRoot, currentWatchRootMetadata); err != nil {
						forceRecreate = true
						watchRootCheckTimer.Reset(watchRestartWait)
						continue
					} else {
						watch = w
						eventPaths = w.eventPaths
					}
				}

				// Since the root changed (or we're in a forced recreate,
				// probably due to or resulting in missed events), we'll also
				// want to send an event.
				select {
				case events <- struct{}{}:
				default:
				}
			}

			// Update parameters.
			watchRootExists = watchRootCurrentlyExists
			watchRootMetadata = currentWatchRootMetadata

			// Reset the timer and continue polling.
			watchRootCheckTimer.Reset(watchRootParameterPollingInterval)
		}
	}
}
