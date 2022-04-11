package watching

import (
	"errors"
	"time"
)

var (
	// ErrWatchTerminated indicates that a watcher has been terminated.
	ErrWatchTerminated = errors.New("watch terminated")
	// ErrTooManyPendingPaths indicates that too many paths were coalesced.
	ErrTooManyPendingPaths = errors.New("too many pending paths")
)

// Filter is a callback type that can be used to exclude paths from being
// returned by a watcher. It accepts a path and returns true if that path should
// be ignored and excluded from events.
type Filter func(string) bool

const (
	// watchCoalescingWindow is the time window for event coalescing.
	watchCoalescingWindow = 20 * time.Millisecond
	// watchCoalescingMaximumPendingPaths is the maximum number of paths that
	// will be allowed in a pending coalesced event.
	watchCoalescingMaximumPendingPaths = 128
)
