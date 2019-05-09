// +build darwin,cgo

package watching

import (
	"context"
	"os"
	"time"

	"github.com/havoc-io/fsevents"
)

const (
	// fseventsCoalescingLatency is the coalescing latency to use with FSEvents
	// itself.
	fseventsCoalescingLatency = 10 * time.Millisecond

	// fseventsFlags are the flags to use for recursive FSEvents watchers.
	fseventsFlags = fsevents.WatchRoot | fsevents.FileEvents
)

// recursiveWatch represents a recursive native watch.
type recursiveWatch struct {
	// eventStream is the underlying event stream.
	eventStream *fsevents.EventStream
	// forwardingCancel cancels event path forwarding from the underlying
	// event stream.
	forwardingCancel context.CancelFunc
	// eventPaths is a channel to which event paths are forwarded from the
	// watcher.
	eventPaths chan string
}

// newRecursiveWatch establishes a new recursive watch.
func newRecursiveWatch(path string, info os.FileInfo) (*recursiveWatch, error) {
	// Create the raw event channel.
	rawEvents := make(chan []fsevents.Event, watchNativeEventsBufferSize)

	// Create the event stream.
	eventStream := &fsevents.EventStream{
		Events:  rawEvents,
		Paths:   []string{path},
		Latency: fseventsCoalescingLatency,
		Flags:   fseventsFlags,
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
			case es, ok := <-rawEvents:
				if !ok {
					break Forwarding
				}
				for _, e := range es {
					select {
					case eventPaths <- e.Path:
					default:
					}
				}
			}
		}
		close(eventPaths)
	}()

	// Start watching.
	eventStream.Start()

	// Done.
	return &recursiveWatch{
		eventStream:      eventStream,
		forwardingCancel: forwardingCancel,
		eventPaths:       eventPaths,
	}, nil
}

// stop terminates the watch.
func (w *recursiveWatch) stop() {
	// Stop the underlying event stream.
	w.eventStream.Stop()

	// Cancel event forwarding.
	w.forwardingCancel()
}
