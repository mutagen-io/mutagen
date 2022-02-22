//go:build (!darwin && !windows) || (darwin && !cgo)

package watching

const (
	// RecursiveWatchingSupported indicates whether or not the current platform
	// supports native recursive watching.
	RecursiveWatchingSupported = false
)

// NewRecursiveWatcher creates a new recursive watcher on platforms that support
// native recursive watching. This platform does not support recursive watching
// and this function will panic if called.
func NewRecursiveWatcher(_ string) (RecursiveWatcher, error) {
	panic("recursive watching not supported on this platform")
}
