package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	environmentpkg "github.com/mutagen-io/mutagen/pkg/environment"
)

// infoResponse is a structure that can be used to decode JSONified output from
// the docker info command.
type infoResponse struct {
	// ServerErrors are any errors that occurred while connecting to the Docker
	// daemon.
	ServerErrors []string `json:"ServerErrors"`
	// ID is the Docker daemon identifier.
	ID string `json:"ID"`
	// OSType is the Docker daemon OS. Its value will be a GOOS value.
	OSType string `json:"OSType"`
}

// GetDaemonMetadata uses the Docker CLI to query the target Docker daemon
// OS and identifier via its /info endpoint. The provided connection flags and
// environment variables are used when executing the docker info command. If
// environment is nil, then the current process' environment will be used.
func GetDaemonMetadata(daemonFlags DaemonConnectionFlags, environment map[string]string) (string, string, error) {
	// Set up flags and arguments to dump server information in JSON format.
	var arguments []string
	arguments = append(arguments, daemonFlags.ToFlags()...)
	arguments = append(arguments, "info", "--format", "{{json .}}")

	// Set up the command.
	command, err := Command(context.Background(), arguments...)
	if err != nil {
		return "", "", fmt.Errorf("unable to set up Docker invocation: %w", err)
	}

	// Set the command environment.
	command.Env = environmentpkg.FromMap(environment)

	// Run the command.
	output, err := command.Output()
	if err != nil {
		return "", "", fmt.Errorf("docker info command failed: %w", err)
	}

	// Perform JSON decoding.
	var info infoResponse
	if err := json.Unmarshal(output, &info); err != nil {
		return "", "", fmt.Errorf("unable to decode JSON response: %w", err)
	}

	// Handle server connection errors.
	if len(info.ServerErrors) > 0 {
		return "", "", errors.New(info.ServerErrors[0])
	}

	// Success.
	return info.OSType, info.ID, nil
}
