package filesystem

import (
	"os"
)

// Mode is an opaque type representing a file mode. It is guaranteed to be
// convertable to a uint32 value. On Windows sytems, it is provided by the os
// package's FileMode implementation.
type Mode os.FileMode

const (
	// ModeTypeMask is a bit mask that isolates type information. After masking,
	// the resulting value can be compared with any of the ModeType* values
	// (other than ModeTypeMask).
	ModeTypeMask = Mode(os.ModeType)
	// ModeTypeDirectory represents a directory.
	ModeTypeDirectory = Mode(os.ModeDir)
	// ModeTypeFile represents a file.
	ModeTypeFile = Mode(0)
	// ModeTypeSymbolicLink represents a symbolic link.
	ModeTypeSymbolicLink = Mode(os.ModeSymlink)
	// ModePermissionsMask is a bit mask that isolates permission bits.
	ModePermissionsMask = Mode(os.ModePerm)
	// ModePermissionUserRead is the user readable bit.
	ModePermissionUserRead = Mode(0400)
	// ModePermissionUserWrite is the user writable bit.
	ModePermissionUserWrite = Mode(0200)
	// ModePermissionUserExecutable is the user executable bit.
	ModePermissionUserExecutable = Mode(0100)
	// ModePermissionGroupRead is the group readable bit.
	ModePermissionGroupRead = Mode(0040)
	// ModePermissionGroupWrite is the group writable bit.
	ModePermissionGroupWrite = Mode(0020)
	// ModePermissionGroupExecutable is the group executable bit.
	ModePermissionGroupExecutable = Mode(0010)
	// ModePermissionOthersRead is the others readable bit.
	ModePermissionOthersRead = Mode(0004)
	// ModePermissionOthersWrite is the others writable bit.
	ModePermissionOthersWrite = Mode(0002)
	// ModePermissionOthersExecutable is the others executable bit.
	ModePermissionOthersExecutable = Mode(0001)
)
