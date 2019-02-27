package docker

// dockerCommandForPlatform returns the name of the docker command on Windows
// platforms, which will force resolution via the PATH environment variable.
func dockerCommandForPlatform() (string, error) {
	return "docker", nil
}
