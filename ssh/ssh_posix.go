// +build !windows,!plan9

// TODO: Figure out what to do for Plan 9. It doesn't support Setsid.

package ssh

import (
	"fmt"
	"path/filepath"
	"syscall"

	"github.com/havoc-io/mutagen/filesystem"
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

func controlMasterArguments() []string {
	// Compute the path to the connections directory. If we fail, then we can
	// just continue without ControlMaster support.
	// TODO: If this fails, we can continue, but something is probably very
	// wrong. It'll probably crop up somewhere else, but maybe we should report
	// it here? The current function signature is much nicer though.
	connectionsDirectoryPath, err := filesystem.Mutagen(true, connectionsDirectoryName)
	if err != nil {
		return nil
	}

	// Compute the template for the connection socket path.
	controlPath := filepath.Join(connectionsDirectoryPath, "%r_%h_%p")

	// Create the necessary arguments.
	return []string{
		"-oTCPKeepAlive=yes",
		"-oServerAliveInterval=60",
		"-oControlMaster=auto",
		fmt.Sprintf("-oControlPath=%s", controlPath),
		"-oControlPersist=1h",
	}
}
