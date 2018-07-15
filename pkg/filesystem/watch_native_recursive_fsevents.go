// +build darwin,cgo

package filesystem

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/fsnotify/fsevents"
)

const (
	// fseventsCoalescingLatency is the coalescing latency to use with FSEvents
	// itself.
	fseventsCoalescingLatency = 10 * time.Millisecond

	// fseventsFlags are the flags to use for recursive FSEvents watchers.
	fseventsFlags = fsevents.WatchRoot | fsevents.FileEvents
)

type recursiveWatch struct {
	eventStream      *fsevents.EventStream
	forwardingCancel context.CancelFunc
	eventPaths       chan string
}

func newRecursiveWatch(path string) (*recursiveWatch, error) {
	// Compute the device ID for the path.
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, errors.Wrap(err, "unable to get watch root metadata")
	}
	deviceID, err := DeviceID(info)
	if err != nil {
		return nil, errors.Wrap(err, "unable to extract watch root device ID")
	}

	// Create the raw event channel.
	rawEvents := make(chan []fsevents.Event, watchEventsBufferSize)

	// Create the event stream.
	eventStream := &fsevents.EventStream{
		Events:  rawEvents,
		Paths:   []string{path},
		Latency: fseventsCoalescingLatency,
		Device:  int32(deviceID),
		Flags:   fseventsFlags,
	}

	// Create the event paths channel.
	eventPaths := make(chan string, watchEventsBufferSize)

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

func (w *recursiveWatch) stop() {
	// Stop the underlying event stream.
	w.eventStream.Stop()

	// Cancel event forwarding.
	w.forwardingCancel()
}
