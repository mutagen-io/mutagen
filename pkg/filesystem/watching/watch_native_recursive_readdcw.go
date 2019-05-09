// +build windows

package watching

import (
	"context"
	"os"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem/watching/third_party/winfsnotify"
)

const (
	// winfsnotifyFlags are the flags to use for recursive winfsnotify watches.
	winfsnotifyFlags = winfsnotify.FS_ALL_EVENTS & ^(winfsnotify.FS_ACCESS | winfsnotify.FS_CLOSE)
)

// recursiveWatch represents a recursive native watch.
type recursiveWatch struct {
	// watcher is the underlying watcher.
	watcher *winfsnotify.Watcher
	// forwardingCancel cancels event path forwarding from the underlying
	// watcher.
	forwardingCancel context.CancelFunc
	// eventPaths is a channel to which event paths are forwarded from the
	// watcher.
	eventPaths chan string
}

// newRecursiveWatch establishes a new recursive watch.
func newRecursiveWatch(path string, _ os.FileInfo) (*recursiveWatch, error) {
	// Create the watcher.
	watcher, err := winfsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create watcher")
	}

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
			case e, ok := <-watcher.Event:
				if !ok || e.Mask == winfsnotify.FS_Q_OVERFLOW {
					break Forwarding
				}
				select {
				case eventPaths <- e.Name:
				default:
				}
			}
		}
		close(eventPaths)
	}()

	// Start watching.
	if err := watcher.AddWatch(path, winfsnotifyFlags); err != nil {
		forwardingCancel()
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, errors.Wrap(err, "unable to start watching")
	}

	// Done.
	return &recursiveWatch{
		watcher:          watcher,
		forwardingCancel: forwardingCancel,
		eventPaths:       eventPaths,
	}, nil
}

// stop terminates the watch.
func (w *recursiveWatch) stop() {
	// Stop the underlying event stream.
	// TODO: Should we handle errors here? There's not really anything sane that
	// we can do.
	w.watcher.Close()

	// Cancel forwarding.
	w.forwardingCancel()
}
