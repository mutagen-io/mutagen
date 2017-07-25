package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const (
	// TODO: Should we make this configurable?
	watchPollInterval = 10 * time.Second
)

func fileInfoEqual(first, second os.FileInfo) bool {
	return first.Size() == second.Size() &&
		first.Mode() == second.Mode() &&
		first.ModTime().Equal(second.ModTime())
}

func poll(root string, existing map[string]os.FileInfo) (map[string]os.FileInfo, bool, error) {
	// Create our result map.
	result := make(map[string]os.FileInfo, len(existing))

	// Create a walk visitor.
	changed := false
	visitor := func(path string, info os.FileInfo, err error) error {
		// If there's an error, pass it forward.
		if err != nil {
			return err
		}

		// Insert the entry for this path.
		result[path] = info

		// Compare the entry for this path.
		if previous, ok := existing[path]; !ok {
			changed = true
		} else if !fileInfoEqual(info, previous) {
			changed = true
		}

		// Success.
		return nil
	}

	// Perform the walk. If it fails, don't return a partial map.
	if err := filepath.Walk(root, visitor); err != nil {
		return nil, false, errors.Wrap(err, "unable to perform filesystem walk")
	}

	// Done.
	return result, changed, nil
}

func Watch(context context.Context, root string, events chan struct{}) error {
	// Attempt to use native watching on this path. This will fail if the path
	// can't be watched natively or if the watch is cancelled.
	watchNative(context, root, events)

	// If native watching failed, check (in a non-blocking fashion) if it was
	// due to cancellation. If so, then we don't want to fall back to polling
	// and can save some setup. If native watching failed for some other reason,
	// then we can fall back to polling until cancellation.
	select {
	case <-context.Done():
		return errors.New("watch cancelled")
	default:
	}

	// Create a timer to regular polling.
	timer := time.NewTimer(watchPollInterval)

	// Loop and poll for changes, but watch for cancellation.
	var contents map[string]os.FileInfo
	for {
		select {
		case <-timer.C:
			// Perform a scan. If there's an error or no change, just continue.
			// We have to assume that errors here are due to concurrent
			// modifications - there's not much we can do to handle them.
			newContents, changed, err := poll(root, contents)
			if err != nil || !changed {
				continue
			}

			// Store the new contents.
			contents = newContents

			// Forward the event in a non-blocking fashion.
			select {
			case events <- struct{}{}:
			default:
			}

			// Reset the timer.
			timer.Reset(watchPollInterval)
		case <-context.Done():
			return errors.New("watch cancelled")
		}
	}
}
