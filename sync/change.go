package sync

type Change struct {
	Path string
	Old  *Entry
	New  *Entry
}

func (c Change) String() string {
	// TODO: Classify the change based on Old/New and provide a more detailed
	// representation.
	return c.Path
}
