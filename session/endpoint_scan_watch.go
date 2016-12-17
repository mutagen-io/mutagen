// +build windows darwin,cgo

package session

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/rjeczalik/notify"
)

const (
	scanPollInterval      = 30 * time.Second
	watchEventsBufferSize = 10
	coalescingWindow      = 250 * time.Millisecond
)

func watch(context context.Context, root string, events chan struct{}) error {
	// Create a watch notifications channel.
	notifications := make(chan notify.EventInfo, watchEventsBufferSize)

	// Create a timer that we can use to coalesce events. It will be created
	// running, so make sure to stop it and consume its first event, if any.
	timer := time.NewTimer(coalescingWindow)
	if !timer.Stop() {
		<-timer.C
	}

	// Compute the parent directory of root. We watch this because (a) we may
	// have a file root, (b) the root may not exist yet, and (c) if you delete
	// the watch root it isn't seen on all platforms. Of course, if the parent
	// of root is deleted, we won't see that either here, but we'll eventually
	// see it in polling.
	parent := filepath.Dir(root)

	// Create a recursive watch. Ensure that it's cancelled when we're done.
	watchPath := fmt.Sprintf("%s/...", parent)
	if err := notify.Watch(watchPath, notifications, notify.All); err != nil {
		return errors.Wrap(err, "unable to create watcher")
	}
	defer notify.Stop(notifications)

	// Poll for the next notification, coalescing, or cancellation.
	for {
		select {
		case notification := <-notifications:
			// Only process notifications that match our target. This test might
			// be a bit fragile, but it should be okay since we normalize our
			// root path. If we receive a match, reset the coalescing timer.
			if strings.HasPrefix(notification.Path(), root) {
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(coalescingWindow)
			}
		case <-timer.C:
			// Forward a coalesced event in a non-blocking manner.
			select {
			case events <- struct{}{}:
			default:
			}
		case <-context.Done():
			return errors.New("cancelled")
		}
	}
}
