package watching

import (
	"context"
	"os"

	"github.com/pkg/errors"

	"github.com/golang/groupcache/lru"

	"github.com/havoc-io/mutagen/pkg/filesystem/watching/internal/third_party/notify"
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

// NonRecursiveMRUWatcher represents a non-recursive native watcher that can
// watch multiple paths and evict old watches on an LRU-basis.
type NonRecursiveMRUWatcher struct {
	// Errors is a buffer channel (with a capacity of one) that will populated
	// with the first internal error that occurs in the watcher. If an error
	// occurs, the watcher should be stopped.
	Errors chan error
	// forwardingCancel cancels event path forwarding from the underlying
	// watcher.
	forwardingCancel context.CancelFunc
	// watcher is the underlying watcher.
	watcher notify.Watcher
	// evictor is the LRU-evicting cache that manages the watch eviction
	// process.
	evictor *lru.Cache
}

// NewNonRecursiveMRUWatcher creates a new non-recursive watcher that will emit
// event paths on the specified events channel. The maximumWatcher argument sets
// the maximum number of watches allowed to exist before LRU-eviction occurs. If
// maximumWatches is 0, a default limit will be used. This function panics if
// the events channel is not buffered or the maximumWatches parameter is
// negative. Unlike recursive watching functions, event paths are not normalized
// to be root-relative, and delivery is on a best-effort basis (i.e. events will
// be dropped if not handled promptly).
func NewNonRecursiveMRUWatcher(events chan string, maximumWatches int) (*NonRecursiveMRUWatcher, error) {
	// Ensure that the events channel is buffered.
	if cap(events) < 1 {
		panic("events channel should be buffered")
	}

	// Validate the maximum watch count and use a default if necessary.
	if maximumWatches < 0 {
		panic("maximum watches negative")
	} else if maximumWatches == 0 {
		maximumWatches = inotifyDefaultMaximumWatches
	}

	// Create the errors channel.
	watchErrors := make(chan error, 1)

	// Create the raw event channel.
	rawEvents := make(chan notify.EventInfo, inotifyChannelCapacity)

	// Start a cancellable Goroutine to extract and forward paths.
	forwardingContext, forwardingCancel := context.WithCancel(context.Background())
	go func() {
		// Track any forwarding errors
		var forwardingError error

		// Perform forwarding.
	Forwarding:
		for {
			select {
			case <-forwardingContext.Done():
				forwardingError = errors.New("forwarding cancelled")
				break Forwarding
			case e, ok := <-rawEvents:
				if !ok {
					forwardingError = errors.New("raw events channel closed")
					break Forwarding
				}
				select {
				case events <- e.Path():
				default:
				}
			}
		}

		// Register any fowarding error.
		if forwardingError != nil {
			select {
			case watchErrors <- forwardingError:
			default:
			}
		}
	}()

	// Create a watcher.
	watcher := notify.NewWatcher(rawEvents)

	// Create an LRU cache to serve as our watch evictor. The keys in this cache
	// are paths and the values are simply 0-valued integers. We only use it to
	// track path watching order and to manage the eviction process.
	evictor := lru.New(maximumWatches)
	evictor.OnEvicted = func(key lru.Key, _ interface{}) {
		if path, ok := key.(string); !ok {
			panic("invalid key type in watch path cache")
		} else {
			if err := watcher.Unwatch(path); err != nil {
				select {
				case watchErrors <- errors.Wrap(err, "unwatch error"):
				default:
				}
			}
		}
	}

	// Done.
	return &NonRecursiveMRUWatcher{
		Errors:           watchErrors,
		forwardingCancel: forwardingCancel,
		watcher:          watcher,
		evictor:          evictor,
	}, nil
}

// Watch adds a watch path to the watcher. Any watching error will be reported
// through the errors channel.
func (w *NonRecursiveMRUWatcher) Watch(path string) {
	// If this path is already watched, then evict it first so that we can
	// establish a clean watch and so that the new watch becomes the
	// most-recently-added record in the evictor.
	if _, ok := w.evictor.Get(path); ok {
		w.evictor.Remove(path)
	}

	// Start the watch. If it fails due to a non-existence error, then we can
	// just avoid adding it. If it fails for any other reason, report the error
	// via the errors channel, otherwise record the watch in the cache. We could
	// return the error directly, but for consistency with the rest of the code
	// (and to make error monitoring easier), we report it via the errors
	// channel.
	err := w.watcher.Watch(
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
			case w.Errors <- err:
			default:
			}
		}
	} else {
		w.evictor.Add(path, 0)
	}
}

// Unwatch removes a watch path from the watcher. Any unwatching error will be
// reported through the errors channel.
func (w *NonRecursiveMRUWatcher) Unwatch(path string) {
	// Evict via the evictor. This is a no-op if the path isn't watched.
	w.evictor.Remove(path)
}

// Stop terminates all watches.
func (w *NonRecursiveMRUWatcher) Stop() {
	// Stop the underlying event stream.
	// TODO: Should we handle errors here? There's not really anything sane that
	// we can do.
	w.watcher.Close()

	// Cancel forwarding.
	w.forwardingCancel()
}
