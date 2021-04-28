package transport

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// ProcessAttributes returns the process attributes to use for starting
// transport processes from the daemon.
func ProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_CONSOLE,
	}
}
