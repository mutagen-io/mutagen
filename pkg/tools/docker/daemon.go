package docker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	environmentpkg "github.com/mutagen-io/mutagen/pkg/environment"
)

const (
	// daemonIdentifierInfoFormat is the formatting string to use with the
	// docker info command to determine the daemon identifier. The check for a
	// lack of server errors is necessary because the ID field is referenced
	// through an embedded struct pointer in the template context and this
	// pointer will be nil if there are server errors (in which case accessing
	// the ID field will cause a panic in the Docker CLI).
	daemonIdentifierInfoFormat = `{{if not .ServerErrors}}{{.ID}}{{end}}`
)

// GetDaemonIdentifier uses the Docker CLI to query the target Docker daemon
// identifier via its /info endpoint. The provided connection flags and
// environment variables are used when executing the docker info command. If
// environment is nil, then the current process' environment will be used.
func GetDaemonIdentifier(flags DaemonConnectionFlags, environment map[string]string) (string, error) {
	// Set up flags and arguments.
	var arguments []string
	arguments = append(arguments, flags.ToFlags()...)
	arguments = append(arguments, "info", "--format", daemonIdentifierInfoFormat)

	// Set up the command.
	command, err := Command(context.Background(), arguments...)
	if err != nil {
		return "", fmt.Errorf("unable to set up Docker invocation: %w", err)
	}

	// Set the command environment.
	command.Env = environmentpkg.FromMap(environment)

	// Run the command.
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("docker info command failed: %w", err)
	}

	// Verify that the output is UTF-8.
	if !utf8.Valid(output) {
		return "", errors.New("docker info command returned non-UTF-8 output")
	}

	// Extract the identifier.
	return strings.TrimSpace(string(output)), nil
}
