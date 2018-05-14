// +build windows darwin,cgo

package filesystem

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/rjeczalik/notify"
)

const (
	watchEventsBufferSize = 25
	coalescingWindow      = 250 * time.Millisecond
)

// isParentOrSelf returns true if and only if parent is a parent path of child
// or equal to child. It is designed to be cheap at the (potential) cost of
// correctness, but it is only designed for internal usage with file
// notifications, so this is probably acceptable. It assumes UTF-8 encoding.
func isParentOrSelf(parent, child string) bool {
	parentLength := len(parent)
	childLength := len(child)
	if childLength < parentLength {
		return false
	} else if parent != child[:parentLength] {
		return false
	} else if childLength > parentLength {
		return os.IsPathSeparator(child[parentLength])
	}
	return true
}

func watchNative(context context.Context, root string, events chan struct{}) error {
	// We only support watching for roots that are descendants of the home
	// directory or the home directory itself.
	if !isParentOrSelf(HomeDirectory, root) {
		return errors.New("root is not a subpath of home")
	}

	// Create a watch events channel.
	nativeEvents := make(chan notify.EventInfo, watchEventsBufferSize)

	// Create a timer that we can use to coalesce events. It will be created
	// running, so make sure to stop it and consume its first event, if any.
	timer := time.NewTimer(coalescingWindow)
	if !timer.Stop() {
		<-timer.C
	}

	// Create a recursive watch on the home directory. Ensure that it's stopped
	// when we're done.
	watchPath := fmt.Sprintf("%s/...", HomeDirectory)
	if err := notify.Watch(watchPath, nativeEvents, notify.All); err != nil {
		return errors.Wrap(err, "unable to create watcher")
	}
	defer notify.Stop(nativeEvents)

	// Poll for the next event, coalesced event, or cancellation. When we
	// receive an event that matches our watch root, we reset the coalescing
	// timer. When the coalescing timer fires, we send an event in a
	// non-blocking fashion. If we're cancelled, we return.
	for {
		select {
		case e := <-nativeEvents:
			if isParentOrSelf(root, e.Path()) {
				if !timer.Stop() {
					// We have to do a non-blocking drain here because we don't
					// know if a false return value from Stop indicates that we
					// didn't stop the timer before it expired or that the timer
					// simply wasn't running (see the definition of Stop's
					// return value in the Go documentation). This differs from
					// above where we know the timer was running and that there
					// will be a value to drain if it's expired. What we're
					// doing here is fine, it just differs from the
					// documentation's example that's designed for cases where
					// you know the timer was running, but it'll still drain any
					// value that's present, there's no race condition or
					// anything.
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(coalescingWindow)
			}
		case <-timer.C:
			select {
			case events <- struct{}{}:
			default:
			}
		case <-context.Done():
			return errors.New("watch cancelled")
		}
	}
}
