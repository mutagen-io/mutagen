package docker

import (
	"strings"

	"github.com/mutagen-io/mutagen/pkg/environment"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// setDockerVariables updates a base environment specification by setting Docker
// environment variables to match those from a Docker URL. Any known Docker
// environment variables that aren't present in the URL's variables are filtered
// from the environment.
func setDockerVariables(base []string, variables map[string]string) []string {
	// Convert the base environment to a map for easier manipulation.
	result := environment.ToMap(base)

	// Populate Docker environment variables. If a given variable wasn't stored
	// in the URL, then remove it from the environment.
	for _, variable := range url.DockerEnvironmentVariables {
		if value, ok := variables[variable]; ok {
			result[variable] = value
		} else {
			delete(result, variable)
		}
	}

	// Done.
	return environment.FromMap(result)
}

// findEnviromentVariable parses an environment variable block of the form
// VAR1=value1[\r]\nVAR2=value2[\r]\n... and searches for the specified
// variable.
func findEnviromentVariable(block, variable string) (string, bool) {
	// Parse the environment variable block.
	parsed := environment.ParseBlock(block)

	// Search through the environment for the specified variable.
	for _, line := range parsed {
		if strings.HasPrefix(line, variable+"=") {
			return line[len(variable)+1:], true
		}
	}

	// No match.
	return "", false
}
