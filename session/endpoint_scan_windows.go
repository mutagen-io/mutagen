// +build ignore

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
)

func watch(context context.Context, root string, events chan struct{}) error {
	// Create a watch notifications channel.
	notifications := make(chan notify.EventInfo, watchEventsBufferSize)

	// Compute the parent directory of root. We watch this because (a) we may
	// have a file root and Windows doesn't support watching a file directly
	// (only a directory), (b) the root may not exist yet, and (c) deletion of
	// the watch root isn't seen on Windows. Of course, if the parent of root is
	// deleted, we won't see that either here, but we'll eventually see it in
	// polling.
	parent := filepath.Dir(root)

	// Create a recursive watch. Ensure that it's cancelled when we're done.
	watchPath := fmt.Sprintf("%s/...", parent)
	if err := notify.Watch(watchPath, notifications, notify.All); err != nil {
		return errors.Wrap(err, "unable to create watcher")
	}
	defer notify.Stop(notifications)

	// Poll for the next notification or cancellation.
	for {
		select {
		case notification := <-notifications:
			// Only process notifications that match our target. This test might
			// be a bit fragile, but it should be okay since we normalize our
			// root path and notification paths are supposed to be absolute and
			// real. If we receive a match, forward the event in a non-blocking
			// manner.
			if strings.HasPrefix(notification.Path(), root) {
				select {
				case events <- struct{}{}:
				default:
				}
			}
		case <-context.Done():
			return errors.New("watch cancelled")
		}
	}
}
