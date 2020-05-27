package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	environmentpkg "github.com/mutagen-io/mutagen/pkg/environment"
)

// DaemonMetadata encodes Docker daemon metadata.
type DaemonMetadata struct {
	// Identifier is the Docker daemon identifier.
	Identifier string `json:"ID"`
	// Platform is the Docker daemon OS. Its value will be a GOOS value.
	Platform string `json:"OSType"`
}

// infoResponse is a structure that can be used to decode JSONified output from
// the docker info command.
type infoResponse struct {
	// ServerErrors are any errors that occurred while connecting to the Docker
	// daemon.
	ServerErrors []string `json:"ServerErrors"`
	// DaemonMetadata is the embedded Docker daemon metadata.
	DaemonMetadata
}

// GetDaemonMetadata uses the Docker CLI to query the target Docker daemon
// OS and identifier via its /info endpoint. The provided connection flags and
// environment variables are used when executing the docker info command. If
// environment is nil, then the current process' environment will be used.
func GetDaemonMetadata(daemonFlags DaemonConnectionFlags, environment map[string]string) (DaemonMetadata, error) {
	// Set up flags and arguments to dump server information in JSON format.
	var arguments []string
	arguments = append(arguments, daemonFlags.ToFlags()...)
	arguments = append(arguments, "info", "--format", "{{json .}}")

	// Set up the command.
	command, err := Command(context.Background(), arguments...)
	if err != nil {
		return DaemonMetadata{}, fmt.Errorf("unable to set up Docker invocation: %w", err)
	}

	// Set the command environment.
	command.Env = environmentpkg.FromMap(environment)

	// Run the command.
	output, err := command.Output()
	if err != nil {
		return DaemonMetadata{}, fmt.Errorf("docker info command failed: %w", err)
	}

	// Perform JSON decoding.
	var info infoResponse
	if err := json.Unmarshal(output, &info); err != nil {
		return DaemonMetadata{}, fmt.Errorf("unable to decode JSON response: %w", err)
	}

	// Handle server connection errors.
	if len(info.ServerErrors) > 0 {
		return DaemonMetadata{}, errors.New(info.ServerErrors[0])
	}

	// Success.
	return info.DaemonMetadata, nil
}
