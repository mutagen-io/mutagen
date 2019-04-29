package docker

import (
	"fmt"
	"strings"

	"github.com/havoc-io/mutagen/pkg/url"
)

// setDockerVariables sets all Docker environment variables to their values
// frozen into the URL.
func setDockerVariables(environment []string, remote *url.URL) []string {
	// Populate all Docker environment variables, overriding any set in the base
	// environment.
	for _, variable := range url.DockerEnvironmentVariables {
		environment = append(environment,
			fmt.Sprintf("%s=%s", variable, remote.Environment[variable]),
		)
	}

	// Done.
	return environment
}

// findEnviromentVariable parses an environment variable block of the form
// VAR1=value1[\r]\nVAR2=value2[\r]\n... and searches for the specified
// variable.
func findEnviromentVariable(outputBlock, variable string) (string, bool) {
	// Parse the output block into a series of VAR=value lines. First we replace
	// \r\n instances with \n, in case the block comes from Windows, trim any
	// outer whitespace (e.g. trailing newlines), and then split on newlines.
	// TODO: We might be able to switch this function to use a bufio.Scanner for
	// greater efficiency.
	outputBlock = strings.ReplaceAll(outputBlock, "\r\n", "\n")
	outputBlock = strings.TrimSpace(outputBlock)
	environment := strings.Split(outputBlock, "\n")

	// Search through the environment for the specified variable.
	for _, line := range environment {
		if strings.HasPrefix(line, variable+"=") {
			return line[len(variable)+1:], true
		}
	}

	// No match.
	return "", false
}
