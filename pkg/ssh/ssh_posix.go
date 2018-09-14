// +build !windows

package ssh

// scpCommandName returns the name of or path to the scp command.
func scpCommandName() (string, error) {
	return "scp", nil
}

// sshCommandName returns the name of or path to the ssh command.
func sshCommandName() (string, error) {
	return "ssh", nil
}
