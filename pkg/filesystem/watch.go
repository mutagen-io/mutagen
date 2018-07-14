package filesystem

import (
	"context"

	"github.com/pkg/errors"
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

	// If we're in portable watch mode, attempt to watch using a native
	// mechanism.
	if mode == WatchMode_WatchPortable {
		watchNative(context, root, events)
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

	// Fall back to pure polling.
	watchPoll(context, root, events, pollInterval)
}
