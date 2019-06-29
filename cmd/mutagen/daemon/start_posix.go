// +build !windows,!plan9

// TODO: Figure out what to do for Plan 9. It doesn't support Setsid.

package daemon

import (
	"syscall"
)

var daemonProcessAttributes = &syscall.SysProcAttr{
	Setsid: true,
}
