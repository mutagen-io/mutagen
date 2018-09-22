package url

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	// dockerURLPrefix is the lowercase version of the Docker URL prefix.
	dockerURLPrefix = "docker://"

	// DockerHostEnvironmentVariable is the name of the DOCKER_HOST environment
	// variable.
	DockerHostEnvironmentVariable = "DOCKER_HOST"
	// DockerTLSVerifyEnvironmentVariable is the name of the DOCKER_TLS_VERIFY
	// environment variable.
	DockerTLSVerifyEnvironmentVariable = "DOCKER_TLS_VERIFY"
	// DockerCertPathEnvironmentVariable is the name of the DOCKER_CERT_PATH
	// environment variable.
	DockerCertPathEnvironmentVariable = "DOCKER_CERT_PATH"
)

// DockerEnvironmentVariables is a list of Docker environment variables that
// should be locked in to the URL at parse time.
var DockerEnvironmentVariables = []string{
	DockerHostEnvironmentVariable,
	DockerTLSVerifyEnvironmentVariable,
	DockerCertPathEnvironmentVariable,
}

// isDockerURL checks whether or not a URL is a Docker URL.
func isDockerURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(raw), dockerURLPrefix)
}

// parseDocker parses a Docker URL.
func parseDocker(raw string, alpha bool) (*URL, error) {
	// Strip off the prefix.
	raw = raw[len(dockerURLPrefix):]

	// Parse off the username. If we hit a '/', then we've reached the end of a
	// container specification and there was no username. Similarly, if we hit
	// the end of the string without seeing an '@', then there's also no
	// username specified. Ideally we'd want also to break on any character that
	// isn't allowed in a username, but that isn't well-defined, even for POSIX
	// (it's effectively determined by a configurable regular expression -
	// NAME_REGEX).
	var username string
	for i, r := range raw {
		if r == '/' {
			break
		} else if r == '@' {
			username = raw[:i]
			raw = raw[i+1:]
			break
		}
	}

	// Split what remains into the container and the path. Ideally we'd want to
	// be a bit more stringent here about what characters we accept in container
	// names, potentially breaking early with an error if we see a "disallowed"
	// character, but we're better off just allowing Docker to reject container
	// names that it doesn't like.
	var container, path string
	for i, r := range raw {
		if r == '/' {
			container = raw[:i]
			path = raw[i:]
			break
		}
	}
	if container == "" {
		return nil, errors.New("empty container name")
	} else if path == "" {
		return nil, errors.New("empty path")
	}

	// If the path starts with "/~", then we assume that it's supposed to be a
	// home-directory-relative path and remove the slash. At this point we
	// already know that the path starts with "/" since we retained that as part
	// of the path in the split operation above.
	if len(path) > 1 && path[1] == '~' {
		path = path[1:]
	}

	// If the path is of the form "/" + Windows path, then assume it's supposed
	// to be a Windows path. This is a heuristic, but a reasonable one. We do
	// this on all systems (not just on Windows as with SSH URLs) because users
	// can connect to Windows containers from non-Windows systems. At this point
	// we already know that the path starts with "/" since we retained that as
	// part of the path in the split operation above.
	if isWindowsPath(path[1:]) {
		path = path[1:]
	}

	// Loop over and record the values for the Docker environment variables that
	// we need to preserve. For the variables in question, Docker treats an
	// empty value the same as an unspecified value, so we always store
	// something for each variable, even if it's just an empty string to
	// indicate that its value was empty or unspecified.
	// TODO: I'm a little concerned that Docker may eventually add environment
	// variables where an empty value is not the same as an unspecified value,
	// but we'll cross that bridge when we come to it.
	environment := make(map[string]string, len(DockerEnvironmentVariables))
	for _, variable := range DockerEnvironmentVariables {
		value, _ := getEnvironmentVariable(variable, alpha)
		environment[variable] = value
	}

	// Success.
	return &URL{
		Protocol:    Protocol_Docker,
		Username:    username,
		Hostname:    container,
		Path:        path,
		Environment: environment,
	}, nil
}
