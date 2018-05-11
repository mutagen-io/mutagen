// +build !windows

package filesystem

import (
	"os"
	"syscall"
)

func isCrossDeviceError(err error) bool {
	if linkErr, ok := err.(*os.LinkError); !ok {
		return false
	} else {
		return linkErr.Err == syscall.EXDEV
	}
}
