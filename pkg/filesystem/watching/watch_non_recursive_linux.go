package watching

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

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
	events chan string
	// errors is the error delivery channel.
	errors chan error
	// cancel is the run loop cancellation function.
	cancel context.CancelFunc
	// done is the run loop completion signaling mechanism.
	done sync.WaitGroup
}

// NewNonRecursiveWatcher creates a new inotify-based non-recursive watcher.
func NewNonRecursiveWatcher() (NonRecursiveWatcher, error) {
	// Create the raw event channel.
	rawEvents := make(chan notify.EventInfo, inotifyChannelCapacity)

	// Create a context to regulate the watcher's run loop.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the watcher.
	watcher := &nonRecursiveWatcher{
		watch:   notify.NewWatcher(rawEvents),
		evictor: lru.New(inotifyDefaultMaximumWatches),
		events:  make(chan string),
		errors:  make(chan error, 1),
		cancel:  cancel,
	}

	// Set the eviction handler.
	watcher.evictor.OnEvicted = func(key lru.Key, _ any) {
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
		case watcher.errors <- watcher.run(ctx, rawEvents):
		default:
		}
		watcher.done.Done()
	}()

	// Success.
	return watcher, nil
}

// run implements the event processing run loop for nonRecursiveWatcher.
func (w *nonRecursiveWatcher) run(ctx context.Context, rawEvents <-chan notify.EventInfo) error {
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

			// Transmit the path.
			select {
			case w.events <- e.Path():
			case <-ctx.Done():
				return ErrWatchTerminated
			}
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
func (w *nonRecursiveWatcher) Events() <-chan string {
	return w.events
}

// Errors implements NonRecursiveWatcher.Errors.
func (w *nonRecursiveWatcher) Errors() <-chan error {
	return w.errors
}

// Terminate implements NonRecursiveWatcher.Terminate.
func (w *nonRecursiveWatcher) Terminate() error {
	// Signal termination.
	w.cancel()

	// Wait for the run loop to exit.
	w.done.Wait()

	// Terminate the underlying watcher.
	return w.watch.Close()
}
