package filesystem

import (
	"os"
	"syscall"
)

const (
	// ERROR_NOT_SAME_DEVICE is the error code returned by MoveFileEx on Windows
	// when attempting to move a file across devices. This can actually be
	// avoided on Windows by specifying the MOVEFILE_COPY_ALLOWED flag, but Go's
	// standard library doesn't do this (most likely to keep consistency with
	// POSIX, which has no such facility).
	ERROR_NOT_SAME_DEVICE = 0x11
)

func isCrossDeviceError(err error) bool {
	if linkErr, ok := err.(*os.LinkError); !ok {
		return false
	} else if errno, ok := linkErr.Err.(syscall.Errno); !ok {
		return false
	} else {
		return errno == ERROR_NOT_SAME_DEVICE
	}
}
