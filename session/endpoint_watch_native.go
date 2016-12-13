// +build ignore

// +build windows darwin,cgo

package session

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"github.com/rjeczalik/notify"
)

const (
	watchEventsBufferSize = 10
	coalescingWindow      = 250 * time.Millisecond
)

func watch(ctx context.Context, root string, events chan struct{}) error {
	// Create an notifications channel.
	notifications := make(chan notify.EventInfo, watchEventsBufferSize)

	// Create a timer that we can use to coalesce events. It will be created
	// running, so make sure to stop it and consume its first event, if any.
	timer := time.NewTimer(coalescingWindow)
	if !timer.Stop() {
		<-timer.C
	}

	// Create the watcher. We use a manual path join here because it's not clear
	// how other packages will behave with a triple-dot pattern. Ensure that the
	// watcher is stopped when we're done.
	watchPath := fmt.Sprintf("%s/...", root)
	if err := notify.Watch(watchPath, notifications, notify.All); err != nil {
		return errors.Wrap(err, "unable to start watch")
	}
	defer notify.Stop(notifications)

	// Poll for the next notification, coalescing, or cancellation.
	for {
		select {
		case <-notifications:
			// Reset the coalescing timer.
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(coalescingWindow)
		case <-timer.C:
			// Forward a coalesced event in a non-blocking manner.
			select {
			case events <- struct{}{}:
			default:
			}
		case <-ctx.Done():
			// Abort in the event of cancellation.
			return errors.New("watch cancelled")
		}
	}
}
