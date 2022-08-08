package docker

import (
	"golang.org/x/sys/execabs"
)

// commandPathForPlatform searches for the docker command in the user's path.
func commandPathForPlatform() (string, error) {
	return execabs.LookPath("docker")
}
