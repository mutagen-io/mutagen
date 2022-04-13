package watching

// RecursiveWatcher is the interface implemented by recursive filesystem
// watching implementations. It is not safe for concurrent usage, though the
// channels returned by its methods may (and should) be polled simultaneously.
type RecursiveWatcher interface {
	// Events returns a channel that provides the paths of event notifications.
	Events() <-chan string
	// Errors returns a channel that is populated if a watch error occurs. If an
	// error occurs, then the watcher should be terminated. If Terminate is
	// invoked before any other error occurs, then it will be populated by
	// ErrWatchTerminated.
	Errors() <-chan error
	// Terminate terminates all watching operations and releases any resources
	// associated with the watcher.
	Terminate() error
}
