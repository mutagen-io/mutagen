package daemon

import (
	"syscall"

	"github.com/mutagen-io/mutagen/pkg/process"
)

// daemonProcessAttributes are the process attributes to use for the daemon.
var daemonProcessAttributes = &syscall.SysProcAttr{
	CreationFlags: process.DETACHED_PROCESS | syscall.CREATE_NEW_PROCESS_GROUP,
}
