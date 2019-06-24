// +build !windows

package ssh

// sshCommandNameOrPathForPlatform returns the name of the ssh command on POSIX platforms,
// which will force resolution via the PATH environment variable.
func sshCommandNameOrPathForPlatform() (string, error) {
	return "ssh", nil
}

// scpCommandNameOrPathForPlatform returns the name of the scp command on POSIX platforms,
// which will force resolution via the PATH environment variable.
func scpCommandNameOrPathForPlatform() (string, error) {
	return "scp", nil
}
