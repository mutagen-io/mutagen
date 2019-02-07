package filesystem

import (
	"os"
)

// Mode is an opaque type representing a file mode. It is guaranteed to be
// convertable to a uint32 value. On Windows sytems, it is provided by the os
// package's FileMode implementation.
type Mode os.FileMode

const (
	// ModeTypeMask is a bit mask that isolates type information from a Mode.
	// After masking, the resulting value can be compared with any of the
	// ModeType* values (other than ModeTypeMask, of course).
	ModeTypeMask = Mode(os.ModeType)
	// ModeTypeDirectory represents a directory.
	ModeTypeDirectory = Mode(os.ModeDir)
	// ModeTypeFile represents a file.
	ModeTypeFile = Mode(0)
	// ModeTypeSymbolicLink represents a symbolic link.
	ModeTypeSymbolicLink = Mode(os.ModeSymlink)
	// ModePermissionsMask is a bit mask that isolates permission bits from a
	// Mode.
	ModePermissionsMask = Mode(os.ModePerm)
)
