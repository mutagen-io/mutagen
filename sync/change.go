package sync

type Change struct {
	Path string
	Old  *Entry
	New  *Entry
}
