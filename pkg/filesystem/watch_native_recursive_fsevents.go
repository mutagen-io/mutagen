// +build darwin,cgo

package filesystem

import (
	"context"
	"os"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/fsnotify/fsevents"
)

const (
	// fseventsCoalescingLatency is the coalescing latency to use with FSEvents
	// itself.
	fseventsCoalescingLatency = 25 * time.Millisecond

	// fseventsFlags are the flags to use for recursive FSEvents watchers.
	fseventsFlags = fsevents.WatchRoot | fsevents.FileEvents
)

// watchRootParameters specifies the parameters that should be monitored for
// changes when determining whether or not a watch needs to be re-established.
type watchRootParameters struct {
	// deviceID is the device ID.
	deviceID int32
	// inode is the watch root inode.
	inode uint64
}

// probeWatchRoot probes the parameters of the watch root.
func probeWatchRoot(root string) (watchRootParameters, error) {
	if info, err := os.Lstat(root); err != nil {
		if os.IsNotExist(err) {
			return watchRootParameters{}, err
		}
		return watchRootParameters{}, errors.Wrap(err, "unable to grab root metadata")
	} else if stat, ok := info.Sys().(*syscall.Stat_t); !ok {
		return watchRootParameters{}, errors.New("unable to extract raw root metadata")
	} else {
		return watchRootParameters{
			deviceID: stat.Dev,
			inode:    stat.Ino,
		}, nil
	}
}

// watchRootParametersEqual determines whether or not two watch root parameters
// are equal.
func watchRootParametersEqual(first, second watchRootParameters) bool {
	return first.deviceID == second.deviceID &&
		first.inode == second.inode
}

type recursiveWatch struct {
	eventStream      *fsevents.EventStream
	forwardingCancel context.CancelFunc
	eventPaths       chan string
}

func newRecursiveWatch(path string) (*recursiveWatch, error) {
	// Compute the device ID for the path.
	parameters, err := probeWatchRoot(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, errors.Wrap(err, "unable to grab device ID")
	}

	// Create the raw event channel.
	rawEvents := make(chan []fsevents.Event, watchEventsBufferSize)

	// Create the event stream.
	eventStream := &fsevents.EventStream{
		Events:  rawEvents,
		Paths:   []string{path},
		Latency: fseventsCoalescingLatency,
		Device:  parameters.deviceID,
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
