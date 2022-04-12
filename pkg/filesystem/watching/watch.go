package watching

import (
	"errors"
)

var (
	// ErrWatchInternalOverflow indicates that a watcher saw an event buffering
	// overflow in its underlying watching mechanism.
	ErrWatchInternalOverflow = errors.New("internal event overflow")
	// ErrWatchTerminated indicates that a watcher has been terminated.
	ErrWatchTerminated = errors.New("watch terminated")
)
