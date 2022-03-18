package watching

// NonRecursiveWatcher is the interface implemented by non-recursive filesystem
// watching implementations. It is not safe for concurrent usage, though the
// channels returned by its methods may (and should) be polled simultaneously.
// Non-recursive watching implementations operate on a best-effort basis. They
// may choose to automatically evict paths from the watch (e.g. on a LRU-basis)
// and are not guaranteed to return all events. They also return raw event paths
// and do not perform any sort of path normalization or relativization.
type NonRecursiveWatcher interface {
	// Watch adds a path to the list of paths being watched.
	Watch(path string)
	// Unwatch removes a path from the list of paths being watched.
	Unwatch(path string)
	// Events returns a channel that provides coalesced event notifications.
	Events() <-chan map[string]bool
	// Errors returns a channel that is populated if a watch error occurs. If an
	// error occurs, then the watcher should be terminated. If Terminate is
	// invoked before any other error occurs, then it will be populated by
	// ErrWatchTerminated.
	Errors() <-chan error
	// Terminate terminates all watching operations and releases any resources
	// associated with the watcher.
	Terminate() error
}
