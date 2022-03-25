//go:build linux && sspl && fanotify

package watching

import (
	"github.com/mutagen-io/mutagen/sspl/pkg/filesystem/watching/fanotify"
)

// RecursiveWatchingSupported indicates whether or not the current platform
// supports native recursive watching.
var RecursiveWatchingSupported = fanotify.Supported

func init() {
	// Override the fanotify package's error values.
	fanotify.ErrWatchTerminated = ErrWatchTerminated
	fanotify.ErrTooManyPendingPaths = ErrTooManyPendingPaths
}

// NewRecursiveWatcher creates a new fanotify-based recursive watcher using the
// specified target path. It accepts an optional filter function that can be
// used to exclude paths from being returned by the watcher. If filter is nil,
// then no filtering is performed.
func NewRecursiveWatcher(target string, filter Filter) (RecursiveWatcher, error) {
	return fanotify.NewRecursiveWatcher(target, filter)
}
