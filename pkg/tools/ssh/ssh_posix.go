// +build !windows

package ssh

import (
	"os/exec"
)

// sshCommandPathForPlatform searches for the ssh command in the user's path.
func sshCommandPathForPlatform() (string, error) {
	return exec.LookPath("ssh")
}

// scpCommandPathForPlatform searches for the scp command in the user's path.
func scpCommandPathForPlatform() (string, error) {
	return exec.LookPath("scp")
}
