//go:build !windows

package transport

import (
	"syscall"
)

// ProcessAttributes returns the process attributes to use for starting
// transport processes from the daemon.
func ProcessAttributes() *syscall.SysProcAttr {
	return nil
}
