// +build !windows,!darwin

package kubectl

// kubectlCommandForPlatform returns the name of the kubectl command on POSIX
// platforms, which will force resolution via the PATH environment variable.
func kubectlCommandForPlatform() (string, error) {
	return "kubectl", nil
}
