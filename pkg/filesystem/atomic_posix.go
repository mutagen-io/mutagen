// +build !windows

package filesystem

import (
	"os"
	"syscall"
)

// isCrossDeviceError checks whether or not an error returned by os.Rename is
// due to an attempted rename across devices.
func isCrossDeviceError(err error) bool {
	if linkErr, ok := err.(*os.LinkError); !ok {
		return false
	} else {
		return linkErr.Err == syscall.EXDEV
	}
}
