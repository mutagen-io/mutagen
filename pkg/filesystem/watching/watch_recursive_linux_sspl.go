//go:build linux && sspl

package watching

import (
	"github.com/mutagen-io/mutagen/sspl/pkg/filesystem/watching/fanotify"
)

// RecursiveWatchingSupported indicates whether or not the current platform
// supports native recursive watching.
var RecursiveWatchingSupported = fanotify.Supported

func init() {
	// Override the fanotify package's error values.
	fanotify.ErrWatchInternalOverflow = ErrWatchInternalOverflow
	fanotify.ErrWatchTerminated = ErrWatchTerminated
}

// NewRecursiveWatcher creates a new fanotify-based recursive watcher using the
// specified target path.
func NewRecursiveWatcher(target string) (RecursiveWatcher, error) {
	return fanotify.NewRecursiveWatcher(target)
}
