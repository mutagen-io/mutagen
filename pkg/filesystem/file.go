package filesystem

import (
	"io"
)

// ReadableFile is a union of io.Reader, io.Seeker, and io.Closer.
type ReadableFile interface {
	io.Reader
	io.Seeker
	io.Closer
}

// WritableFile is a union of io.Writer and io.Closer.
type WritableFile interface {
	io.Writer
	io.Closer
}
