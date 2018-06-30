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
	watchEventsBufferSize             = 25
	watchCoalescingWindow             = 100 * time.Millisecond
	watchRootParameterPollingInterval = 5 * time.Second
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

func watchNative(context context.Context, root string, events chan struct{}) error {
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
	var exists bool
	var parameters watchRootParameters

	// Set up our initial event paths channel.
	dummyEventPaths := make(chan string)
	eventPaths := dummyEventPaths

	// Create a placeholder for the watch.
	var watch *recursiveWatch

	// Create a timer that we can use to coalesce events. It will be created
	// running, so make sure to stop it and consume its first event, if any.
	coalescingTimer := time.NewTimer(watchCoalescingWindow)
	if !coalescingTimer.Stop() {
		<-coalescingTimer.C
	}

	// Create a timer to watch for changes to the root device ID and/or inode.
	// Start it with a 0 duration so that the first check takes place
	// immediately. Subsequent checks will take place at the normal interval.
	rootCheckTimer := time.NewTimer(0)

	// Create a function to clean up after ourselves.
	defer func() {
		// Cancel any watch.
		if watch != nil {
			watch.stop()
		}

		// Cancel the coalescing timer.
		coalescingTimer.Stop()

		// Cancel the root check timer.
		rootCheckTimer.Stop()
	}()

	// Poll for the next event, coalesced event, or cancellation. When we
	// receive an event that matches our watch root, we reset the coalescing
	// timer. When the coalescing timer fires, we send an event in a
	// non-blocking fashion. If we're cancelled, we return.
	for {
		select {
		case <-context.Done():
			return errors.New("watch cancelled")
		case path := <-eventPaths:
			// NOTE: When using FSEvents, event paths are (a) relative to the
			// device root and (b) fully resolved in terms of symlinks. This
			// means that the isExecutabilityTestPath and
			// isDecompositionTestPath checks will still work but isParentOrSelf
			// will not. Fortunately isParentOrSelf isn't necessary when using
			// FSEvents since we watch the root itself.
			if isExecutabilityTestPath(path) || isDecompositionTestPath(path) {
				continue
			} else if runtime.GOOS == "windows" && !isParentOrSelf(root, path) {
				continue
			} else {
				if !coalescingTimer.Stop() {
					// We have to do a non-blocking drain here because we don't
					// know if a false return value from Stop indicates that we
					// didn't stop the timer before it expired or that the timer
					// simply wasn't running (see the definition of Stop's
					// return value in the Go documentation). This differs from
					// above where we know the timer was running and that there
					// will be a value to drain if it's expired. What we're
					// doing here is fine, it just differs from the
					// documentation's example that's designed for cases where
					// you know the timer was running, but it'll still drain any
					// value that's present, there's no race condition or
					// anything.
					select {
					case <-coalescingTimer.C:
					default:
					}
				}
				coalescingTimer.Reset(watchCoalescingWindow)
			}
		case <-coalescingTimer.C:
			select {
			case events <- struct{}{}:
			default:
			}
		case <-rootCheckTimer.C:
			// Grab watch root parameters.
			var currentlyExists bool
			var currentParameters watchRootParameters
			if p, err := probeWatchRoot(watchRoot); err != nil {
				if !os.IsNotExist(err) {
					return errors.Wrap(err, "unable to probe root device ID and inode")
				}
			} else {
				currentlyExists = true
				currentParameters = p
			}

			// Check if we need to recreate the watcher.
			recreate := exists != currentlyExists ||
				!watchRootParametersEqual(parameters, currentParameters)

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
				// watching fails, then it's entirely possible that the watch
				// root was deleted between the time we saw it and the time we
				// tried to start watching. Unfortunately, we have to assume
				// that's what went wrong, since there's no way to ensure that
				// the notify package returns something checkable with
				// os.IsNotExist. In any case, if there's an error, then just
				// treat things as if we never saw the watch root existing.
				if currentlyExists {
					if w, err := newRecursiveWatch(watchRoot); err != nil {
						if os.IsNotExist(err) {
							currentlyExists = false
							currentParameters = watchRootParameters{}
						} else {
							return errors.Wrap(err, "unable to create recursive watch")
						}
					} else {
						watch = w
						eventPaths = w.eventPaths
					}
				}

				// Since the root changed, we'll also want to send a change
				// notification, because the change may not have been caught by
				// the watcher.
				select {
				case events <- struct{}{}:
				default:
				}
			}

			// Update parameters.
			exists = currentlyExists
			parameters = currentParameters

			// Reset the timer and continue polling.
			rootCheckTimer.Reset(watchRootParameterPollingInterval)
		}
	}
}
