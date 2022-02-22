package watching

import (
	"errors"
	"time"
)

const (
	// watchCoalescingWindow is the time window for event coalescing.
	watchCoalescingWindow = 10 * time.Millisecond
	// watchCoalescingMaximumPendingPaths is the maximum number of paths that
	// will be allowed in a pending coalesced event.
	watchCoalescingMaximumPendingPaths = 10 * 1024
)

var (
	// ErrWatchTerminated indicates that a watcher has been terminated.
	ErrWatchTerminated = errors.New("watch terminated")
	// ErrTooManyPendingPaths indicates that too many paths were coalesced.
	ErrTooManyPendingPaths = errors.New("too many pending paths")
)
