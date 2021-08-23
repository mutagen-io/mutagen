package stream

import (
	"io"
)

// DualModeReader represents a reader that can perform both regular and
// single-byte reads efficiently.
type DualModeReader interface {
	io.ByteReader
	io.Reader
}
