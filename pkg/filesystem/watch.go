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

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (m *WatchMode) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Convert to a VCS mode.
	switch text {
	case "portable":
		*m = WatchMode_WatchPortable
	case "force-poll":
		*m = WatchMode_WatchForcePoll
	default:
		return errors.Errorf("unknown watch mode specification: %s", text)
	}

	// Success.
	return nil
}

// Supported indicates whether or not a particular watch mode is a valid,
// non-default value.
func (m WatchMode) Supported() bool {
	switch m {
	case WatchMode_WatchPortable:
		return true
	case WatchMode_WatchForcePoll:
		return true
	default:
		return false
	}
}

// Description returns a human-readable description of a watch mode.
func (m WatchMode) Description() string {
	switch m {
	case WatchMode_WatchDefault:
		return "Default"
	case WatchMode_WatchPortable:
		return "Portable"
	case WatchMode_WatchForcePoll:
		return "Force Poll"
	default:
		return "Unknown"
	}
}

func fileInfoEqual(first, second os.FileInfo) bool {
	// Compare modes.
	if first.Mode() != second.Mode() {
		return false
	}

	// If we're dealing with directories, don't check size or time. Size doesn't
	// really make sense and modification time will be affected by our
	// executability preservation or Unicode decomposition probe file creation.
	if first.IsDir() {
		return true
	}

	// Compare size and time.
	return first.Size() == second.Size() &&
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

		// If this is an executability preservation or Unicode decomposition
		// test path, ignore it.
		if isExecutabilityTestPath(path) || isDecompositionTestPath(path) {
			return nil
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
	if mode == WatchMode_WatchPortable {
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

	// Compute the polling interval.
	if pollInterval == 0 {
		pollInterval = DefaultPollingInterval
	}
	pollIntervalDuration := time.Duration(pollInterval) * time.Second

	// Create a timer to regulate polling. Start it with a 0 duration so that
	// the first polling takes place immediately. Subsequent pollings will take
	// place at the normal interval.
	timer := time.NewTimer(0)

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
