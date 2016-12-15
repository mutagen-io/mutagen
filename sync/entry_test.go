package sync

// get is the same as Find, except that it panics if it can't find the requested
// entry. It is primarily syntactic sugar for tests.
func (e *Entry) get(name string) *Entry {
	// Try a normal find.
	if entry, ok := e.Find(name); ok {
		return entry
	}

	// If that didn't work, panic.
	panic("failed to locate entry")
}
