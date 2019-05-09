// +build linux

package watching

import (
	"context"

	"github.com/havoc-io/mutagen/pkg/filesystem/watching/third_party/notify"
)

// nonRecursiveWatcher represents a non-recursive native watcher that can watch
// multiple paths.
type nonRecursiveWatcher struct {
	// watcher is the underlying watcher.
	watcher notify.Watcher
	// forwardingCancel cancels event path forwarding from the underlying
	// watcher.
	forwardingCancel context.CancelFunc
	// eventPaths is a channel to which event paths are forwarded from the
	// watcher.
	eventPaths chan string
}

// newNonRecursiveWatcher creates a new non-recursive watcher.
func newNonRecursiveWatcher() (*nonRecursiveWatcher, error) {
	// Create the raw event channel.
	rawEvents := make(chan notify.EventInfo, watchNativeEventsBufferSize)

	// Create the event paths channel.
	eventPaths := make(chan string, watchNativeEventsBufferSize)

	// Start a cancellable Goroutine to extract and forward paths.
	forwardingContext, forwardingCancel := context.WithCancel(context.Background())
	go func() {
	Forwarding:
		for {
			select {
			case <-forwardingContext.Done():
				break Forwarding
			case e, ok := <-rawEvents:
				if !ok {
					break Forwarding
				}
				select {
				case eventPaths <- e.Path():
				default:
				}
			}
		}
		close(eventPaths)
	}()

	// Create a watcher.
	watcher := notify.NewWatcher(rawEvents)

	// Done.
	return &nonRecursiveWatcher{
		watcher:          watcher,
		forwardingCancel: forwardingCancel,
		eventPaths:       eventPaths,
	}, nil
}

// watch adds a watch path to the watcher.
func (w *nonRecursiveWatcher) watch(path string) error {
	return w.watcher.Watch(
		path,
		notify.InModify|notify.InAttrib|
			notify.InCloseWrite|
			notify.InMovedFrom|notify.InMovedTo|
			notify.InCreate|notify.InDelete|
			notify.InDeleteSelf|notify.InMoveSelf,
	)
}

// unwatch removes a watch path from the watcher.
func (w *nonRecursiveWatcher) unwatch(path string) error {
	return w.watcher.Unwatch(path)
}

// stop terminates all watches.
func (w *nonRecursiveWatcher) stop() {
	// Stop the underlying event stream.
	// TODO: Should we handle errors here? There's not really anything sane that
	// we can do.
	w.watcher.Close()

	// Cancel forwarding.
	w.forwardingCancel()
}
