package daemon

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// daemonProcessAttributes are the process attributes to use for the daemon.
var daemonProcessAttributes = &syscall.SysProcAttr{
	CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
}
