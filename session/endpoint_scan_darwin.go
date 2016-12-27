// +build cgo

package session

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/fsnotify/fsevents"
)

const (
	scanPollInterval      = 30 * time.Second
	watchEventsBufferSize = 10
)

func watch(context context.Context, root string, events chan struct{}) error {
	// Create a recursive watch. Ensure that it's cancelled when we're done.
	device, err := fsevents.DeviceForPath(root)
	if err != nil {
		return errors.Wrap(err, "unable to compute device for path")
	}
	stream := &fsevents.EventStream{
		Paths:  []string{root},
		Device: device,
		Flags:  fsevents.FileEvents | fsevents.WatchRoot,
	}
	stream.Start()
	defer stream.Stop()

	// Poll for the next notification or cancellation.
	for {
		select {
		case <-stream.Events:
			// Forward the event in a non-blocking manner.
			select {
			case events <- struct{}{}:
			default:
			}
		case <-context.Done():
			return errors.New("watch cancelled")
		}
	}
}
