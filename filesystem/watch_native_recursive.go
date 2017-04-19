// +build windows darwin,cgo

package filesystem

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/rjeczalik/notify"
)

const (
	watchEventsBufferSize = 25
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
	if !isParentOrSelf(homeDirectory, root) {
		return errors.New("root is not a subpath of home")
	}

	// Create a watch events channel.
	nativeEvents := make(chan notify.EventInfo, watchEventsBufferSize)

	// Create a recursive watch on the home directory. Ensure that it's stopped
	// when we're done.
	watchPath := fmt.Sprintf("%s/...", homeDirectory)
	if err := notify.Watch(watchPath, nativeEvents, notify.All); err != nil {
		return errors.Wrap(err, "unable to create watcher")
	}
	defer notify.Stop(nativeEvents)

	// Poll for the next event or cancellation. We only forward events that
	// match our root, and do so in a non-blocking manner.
	for {
		select {
		case e := <-nativeEvents:
			if isParentOrSelf(root, e.Path()) {
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
