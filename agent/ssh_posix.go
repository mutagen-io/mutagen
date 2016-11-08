// +build !windows,!plan9

// TODO: Figure out what to do for Plan 9. It doesn't support Setsid.

package agent

import (
	"syscall"
)

func scpCommand() (string, error) {
	return "scp", nil
}

func sshCommand() (string, error) {
	return "ssh", nil
}

func processAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// There's also a Noctty field, but it only detaches standard input from
		// the controlling terminal (not standard output or error), and if
		// standard input isn't a terminal, it will fail to launch the process.
		// Setsid might be a little heavy handed since it creates a new process
		// group, but it also properly detaches the process from any controlling
		// terminal, and it's a standard system call, so it seems to be the most
		// robust option.
		Setsid: true,
	}
}
