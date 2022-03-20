package watching

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"

	"github.com/mutagen-io/mutagen/pkg/filesystem/watching/internal/third_party/notify"
)

const (
	// NonRecursiveWatchingSupported indicates whether or not the current
	// platform supports native non-recursive watching.
	NonRecursiveWatchingSupported = true

	// inotifyChannelCapacity is the capacity to use for the internal inotify
	// events channel.
	inotifyChannelCapacity = 50
	// inotifyDefaultMaximumWatches is the default maximum number of inotify
	// watches that will be allowed to exist per-watcher.
	inotifyDefaultMaximumWatches = 50
)

// nonRecursiveWatcher implements NonRecursiveWatcher using inotify, with paths
// evicted on an LRU-basis.
type nonRecursiveWatcher struct {
	// watch is the underlying inotify-based watcher.
	watch notify.Watcher
	// evictor performs LRU-based watch eviction.
	evictor *lru.Cache
	// events is the event delivery channel.
	events chan map[string]bool
	// watchErrors relays watch errors to the run loop.
	watchErrors chan<- error
	// errors is the error delivery channel.
	errors chan error
	// cancel is the run loop cancellation function.
	cancel context.CancelFunc
	// done is the run loop completion signaling mechanism.
	done sync.WaitGroup
}

// NewNonRecursiveWatcher creates a new inotify-based non-recursive watcher. It
// accepts an optional filter function that can be used to exclude paths from
// being returned by the watcher. If filter is nil, then no filtering is
// performed.
func NewNonRecursiveWatcher(filter Filter) (NonRecursiveWatcher, error) {
	// Create the raw event channel.
	rawEvents := make(chan notify.EventInfo, inotifyChannelCapacity)

	// Create a context to regulate the watcher's run loop.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the watcher.
	watcher := &nonRecursiveWatcher{
		watch:   notify.NewWatcher(rawEvents),
		evictor: lru.New(inotifyDefaultMaximumWatches),
		events:  make(chan map[string]bool),
		errors:  make(chan error, 1),
		cancel:  cancel,
	}

	// Set the eviction handler.
	watcher.evictor.OnEvicted = func(key lru.Key, _ interface{}) {
		if path, ok := key.(string); !ok {
			panic("invalid key type in watch path cache")
		} else {
			if err := watcher.watch.Unwatch(path); err != nil {
				select {
				case watcher.errors <- fmt.Errorf("unwatch error: %w", err):
				default:
				}
			}
		}
	}

	// Track run loop termination.
	watcher.done.Add(1)

	// Start the run loop.
	go func() {
		select {
		case watcher.errors <- watcher.run(ctx, rawEvents, filter):
		default:
		}
		watcher.done.Done()
	}()

	// Success.
	return watcher, nil
}

// run implements the event processing run loop for nonRecursiveWatcher.
func (w *nonRecursiveWatcher) run(ctx context.Context, rawEvents <-chan notify.EventInfo, filter Filter) error {
	// Create a coalescing timer, initially stopped and drained, and ensure that
	// it's stopped once we return.
	coalescingTimer := time.NewTimer(0)
	if !coalescingTimer.Stop() {
		<-coalescingTimer.C
	}
	defer coalescingTimer.Stop()

	// Create an empty pending event.
	pending := make(map[string]bool)

	// Create a separate channel variable to track the target events channel. We
	// keep it nil to block transmission until the pending event is non-empty
	// and the coalescing timer has fired.
	var pendingTarget chan<- map[string]bool

	// Loop indefinitely, polling for cancellation and events.
	for {
		select {
		case <-ctx.Done():
			return ErrWatchTerminated
		case e, ok := <-rawEvents:
			// Ensure that the event channel wasn't closed.
			if !ok {
				return errors.New("raw events channel closed")
			}

			// Extract the path.
			path := e.Path()

			// Check if the path should be excluded.
			if filter != nil && filter(path) {
				continue
			}

			// Record the path.
			pending[path] = true

			// Check if we've exceeded the maximum number of allowed pending
			// paths. We're technically allowing ourselves to go one over the
			// limit here, but to avoid that we'd have to check whether or not
			// each path was already in pending before adding it, and that would
			// be expensive. Since this is a purely internal check for the
			// purpose of avoiding excessive memory usage, this small transient
			// overflow is fine.
			if len(pending) > watchCoalescingMaximumPendingPaths {
				return ErrTooManyPendingPaths
			}

			// We may have already had a pending event that was coalesced and
			// ready to be delivered, but now we've seen more changes and we're
			// going to create a new coalescing window, so we'll block event
			// transmission until the new coalescing window is complete.
			pendingTarget = nil

			// Reset the coalesing timer. We don't know if it was already
			// running, so we need to drain it in a non-blocking fashion.
			if !coalescingTimer.Stop() {
				select {
				case <-coalescingTimer.C:
				default:
				}
			}
			coalescingTimer.Reset(watchCoalescingWindow)
		case <-coalescingTimer.C:
			// Set the target events channel to the actual events channel.
			pendingTarget = w.events
		case pendingTarget <- pending:
			// Create a new pending event.
			pending = make(map[string]bool)

			// Block event transmission until the event is non-empty.
			pendingTarget = nil
		}
	}
}

// Watch implements NonRecursiveWatcher.Watch.
func (w *nonRecursiveWatcher) Watch(path string) {
	// Attempt to evict the path if already watched, that way we can establish a
	// clean watch and make the path the most-recently-added record. If the path
	// isn't currently watched, then this is a no-op.
	w.evictor.Remove(path)

	// Start the watch. If it fails due to a non-existence error, then we can
	// just avoid adding it. If it fails for any other reason, then report the
	// error via the errors channel.
	err := w.watch.Watch(
		path,
		notify.InModify|notify.InAttrib|
			notify.InCloseWrite|
			notify.InMovedFrom|notify.InMovedTo|
			notify.InCreate|notify.InDelete|
			notify.InDeleteSelf|notify.InMoveSelf,
	)
	if err != nil {
		if !os.IsNotExist(err) {
			select {
			case w.errors <- fmt.Errorf("watch error: %w", err):
			default:
			}
		}
	} else {
		w.evictor.Add(path, 0)
	}
}

// Unwatch implements NonRecursiveWatcher.Unwatch.
func (w *nonRecursiveWatcher) Unwatch(path string) {
	// Remove the watch via eviction. This is a no-op if the path isn't watched.
	w.evictor.Remove(path)
}

// Events implements NonRecursiveWatcher.Events.
func (w *nonRecursiveWatcher) Events() <-chan map[string]bool {
	return w.events
}

// Errors implements NonRecursiveWatcher.Errors.
func (w *nonRecursiveWatcher) Errors() <-chan error {
	return w.errors
}

// Terminate implements NonRecursiveWatcher.Terminate.
func (w *nonRecursiveWatcher) Terminate() error {
	// Signal cancellation.
	w.cancel()

	// Wait for the run loop to exit.
	w.done.Wait()

	// Terminate the underlying watcher.
	return w.watch.Close()
}
