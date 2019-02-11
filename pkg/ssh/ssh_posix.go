// +build !windows

package ssh

// sshCommandForPlatform returns the name of the ssh command on POSIX platforms,
// which will force resolution via the PATH environment variable.
func sshCommandForPlatform() (string, error) {
	return "ssh", nil
}

// scpCommandForPlatform returns the name of the scp command on POSIX platforms,
// which will force resolution via the PATH environment variable.
func scpCommandForPlatform() (string, error) {
	return "scp", nil
}
