//go:build !windows

package ssh

import (
	"golang.org/x/sys/execabs"
)

// sshCommandPathForPlatform searches for the ssh command in the user's path.
func sshCommandPathForPlatform() (string, error) {
	return execabs.LookPath("ssh")
}

// scpCommandPathForPlatform searches for the scp command in the user's path.
func scpCommandPathForPlatform() (string, error) {
	return execabs.LookPath("scp")
}
