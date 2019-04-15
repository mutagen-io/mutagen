package kubectl

// kubectlCommandForPlatform returns the name of the kubectl command on Windows
// platforms, which will force resolution via the PATH environment variable.
func kubectlCommandForPlatform() (string, error) {
	return "kubectl", nil
}
