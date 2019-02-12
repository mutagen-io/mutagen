// +build !windows

package filesystem

import (
	"golang.org/x/sys/unix"
)

// Mode is an opaque type representing a file mode. It is guaranteed to be
// convertable to a uint32 value. On POSIX sytems, it is the raw underlying file
// mode from the Stat_t structure (as opposed to the os package's FileMode
// implementation).
type Mode uint32

const (
	// ModeTypeMask is a bit mask that isolates type information. After masking,
	// the resulting value can be compared with any of the ModeType* values
	// (other than ModeTypeMask).
	ModeTypeMask = Mode(unix.S_IFMT)
	// ModeTypeDirectory represents a directory.
	ModeTypeDirectory = Mode(unix.S_IFDIR)
	// ModeTypeFile represents a file.
	ModeTypeFile = Mode(unix.S_IFREG)
	// ModeTypeSymbolicLink represents a symbolic link.
	ModeTypeSymbolicLink = Mode(unix.S_IFLNK)
)
