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
	// ModePermissionsMask is a bit mask that isolates permission bits.
	ModePermissionsMask = Mode(unix.S_IRWXU | unix.S_IRWXG | unix.S_IRWXO)
	// ModeExtendedPermissionsMask is a bit mask that isolates permission bits,
	// including extended permission bits (setuid, setgid, and sticky bits). It
	// is only available on POSIX systems.
	ModeExtendedPermissionsMask = ModePermissionsMask | Mode(unix.S_ISUID|unix.S_ISGID|unix.S_ISVTX)
	// ModePermissionUserRead is the user readable bit.
	ModePermissionUserRead = Mode(unix.S_IRUSR)
	// ModePermissionUserWrite is the user writable bit.
	ModePermissionUserWrite = Mode(unix.S_IWUSR)
	// ModePermissionUserExecutable is the user executable bit.
	ModePermissionUserExecutable = Mode(unix.S_IXUSR)
	// ModePermissionGroupRead is the group readable bit.
	ModePermissionGroupRead = Mode(unix.S_IRGRP)
	// ModePermissionGroupWrite is the group writable bit.
	ModePermissionGroupWrite = Mode(unix.S_IWGRP)
	// ModePermissionGroupExecutable is the group executable bit.
	ModePermissionGroupExecutable = Mode(unix.S_IXGRP)
	// ModePermissionOthersRead is the others readable bit.
	ModePermissionOthersRead = Mode(unix.S_IROTH)
	// ModePermissionOthersWrite is the others writable bit.
	ModePermissionOthersWrite = Mode(unix.S_IWOTH)
	// ModePermissionOthersExecutable is the others executable bit.
	ModePermissionOthersExecutable = Mode(unix.S_IXOTH)
)
