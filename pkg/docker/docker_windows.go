package docker

import (
	"os/exec"
)

// commandPathForPlatform searches for the docker command in the user's path.
func commandPathForPlatform() (string, error) {
	return exec.LookPath("docker")
}
