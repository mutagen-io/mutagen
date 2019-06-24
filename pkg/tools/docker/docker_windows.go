package docker

// commandNameOrPathForPlatform returns the name of the docker command on
// Windows platforms, which will force resolution via the PATH environment
// variable.
func commandNameOrPathForPlatform() (string, error) {
	return "docker", nil
}
