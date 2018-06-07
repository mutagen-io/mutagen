package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const (
	// DefaultPollingInterval is the default watch polling interval, in seconds.
	DefaultPollingInterval = 10
)

// NewWatchModeFromString parses a watch mode specification string and returns a
// WatchMode enumeration value.
func NewWatchModeFromString(mode string) (WatchMode, error) {
	switch mode {
	case "recursive-home":
		return WatchMode_RecursiveHome, nil
	case "poll":
		return WatchMode_Poll, nil
	default:
		return WatchMode_Default, errors.Errorf("unknown mode specified: %s", mode)
	}
}

// Supported indicates whether or not a particular watch mode is supported.
func (m WatchMode) Supported() bool {
	switch m {
	case WatchMode_RecursiveHome:
		return true
	case WatchMode_Poll:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a watch mode.
func (m WatchMode) Description() string {
	switch m {
	case WatchMode_Default:
		return "Default"
	case WatchMode_RecursiveHome:
		return "Recursive Home"
	case WatchMode_Poll:
		return "Poll"
	default:
		return "Unknown"
	}
}

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
	rootDoesNotExist := false
	visitor := func(path string, info os.FileInfo, err error) error {
		// If there's an error, then halt walking by returning it. Before doing
		// that though, determine if the error is due to the root not existing.
		// If that's the case, then we can create a valid result (an empty map)
		// as well as determine whether or not there's been a change.
		if err != nil {
			if path == root && os.IsNotExist(err) {
				changed = len(existing) > 0
				rootDoesNotExist = true
			}
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

	// Perform the walk. If it fails, and it's not due to the root not existing,
	// then we can't return a valid result and need to abort.
	if err := filepath.Walk(root, visitor); err != nil && !rootDoesNotExist {
		return nil, false, errors.Wrap(err, "unable to perform filesystem walk")
	}

	// Done.
	return result, changed, nil
}

// TODO: Document that this function closes the events channel when the watch
// is cancelled.
// TODO: Document that this function will always succeed in one way or another
// (it doesn't have any total failure modes) and won't exit until the associated
// context is cancelled.
// TODO: Document that the events channel must be buffered.
func Watch(context context.Context, root string, events chan struct{}, mode WatchMode, pollInterval uint32) {
	// Ensure that the events channel is buffered.
	if cap(events) < 1 {
		panic("watch channel should be buffered")
	}

	// Ensure that the events channel is closed when we're cancelled.
	defer close(events)

	// If we're in a recurisve home watch mode, attempt to watch in that manner.
	// This will be fail if we're on a system without native recursive watching,
	// the root is not a subpath of the home directory, or the watch is
	// cancelled.
	if mode == WatchMode_RecursiveHome {
		watchRecursiveHome(context, root, events)
	}

	// If native watching failed, check (in a non-blocking fashion) if it was
	// due to cancellation. If so, then we don't want to fall back to polling
	// and can save some setup. If native watching failed for some other reason,
	// then we can fall back to polling until cancellation.
	select {
	case <-context.Done():
		return
	default:
	}

	// Create a timer to regular polling.
	if pollInterval == 0 {
		pollInterval = DefaultPollingInterval
	}
	pollIntervalDuration := time.Duration(pollInterval) * time.Second
	timer := time.NewTimer(pollIntervalDuration)

	// Loop and poll for changes, but watch for cancellation.
	var contents map[string]os.FileInfo
	for {
		select {
		case <-timer.C:
			// Perform a scan. If there's an error or no change, then reset the
			// timer and try again. We have to assume that errors here are due
			// to concurrent modifications, so there's not much we can do to
			// handle them.
			// TODO: If we see a certain number of failed polls, we could just
			// fall back to a timer.
			newContents, changed, err := poll(root, contents)
			if err != nil || !changed {
				timer.Reset(pollIntervalDuration)
				continue
			}

			// Store the new contents.
			contents = newContents

			// Forward the event in a non-blocking fashion.
			select {
			case events <- struct{}{}:
			default:
			}

			// Reset the timer and continue polling.
			timer.Reset(pollIntervalDuration)
		case <-context.Done():
			return
		}
	}
}
