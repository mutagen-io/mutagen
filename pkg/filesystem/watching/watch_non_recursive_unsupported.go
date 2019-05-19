// +build !linux

package watching

const (
	// NonRecursiveWatchingSupported indicates whether or not the current
	// platform supports native non-recursive watching.
	NonRecursiveWatchingSupported = false
)

// NonRecursiveMRUWatcher represents a non-recursive native watcher that can
// watch multiple paths and evict old watches on an LRU-basis.
type NonRecursiveMRUWatcher struct {
	// Errors is a buffer channel (with a capacity of one) that will populated
	// with the first internal error that occurs in the watcher. If an error
	// occurs, the watcher should be stopped. It will never be populated on this
	// platform.
	Errors chan error
}

// NewNonRecursiveMRUWatcher creates a new non-recursive watcher that will emit
// event paths on the specified events channel. This function is not implemented
// on this platform and will panic if called.
func NewNonRecursiveMRUWatcher(_ chan string, _ int) (*NonRecursiveMRUWatcher, error) {
	panic("non-recursive watching not supported on this platform")
}

// Watch adds a watch path to the watcher. This method is not implemented on
// this platform and will panic if called.
func (w *NonRecursiveMRUWatcher) Watch(_ string) error {
	panic("non-recursive watching not supported on this platform")
}

// Unwatch removes a watch path from the watcher. This method is not implemented
// on this platform and will panic if called.
func (w *NonRecursiveMRUWatcher) Unwatch(_ string) error {
	panic("non-recursive watching not supported on this platform")
}

// Stop terminates all watches. This method is not implemented on this platform
// and will panic if called.
func (w *NonRecursiveMRUWatcher) Stop() {
	panic("non-recursive watching not supported on this platform")
}
