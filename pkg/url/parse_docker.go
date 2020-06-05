package url

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// dockerURLPrefix is the lowercase version of the Docker URL prefix.
const dockerURLPrefix = "docker://"

// DockerEnvironmentVariables is a list of Docker environment variables that
// should be locked in to Docker URLs at parse time.
var DockerEnvironmentVariables = []string{
	"DOCKER_HOST",
	"DOCKER_TLS_VERIFY",
	"DOCKER_CERT_PATH",
	"DOCKER_CONTEXT",
	"DOCKER_API_VERSION",
}

// isDockerURL checks whether or not a URL is a Docker URL. It requires the
// presence of a Docker protocol prefix.
func isDockerURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(raw), dockerURLPrefix)
}

// parseDocker parses a Docker URL.
func parseDocker(raw string, kind Kind, first bool) (*URL, error) {
	// Strip off the prefix.
	raw = raw[len(dockerURLPrefix):]

	// Determine the character that splits the container name from the path or
	// forwarding endpoint component.
	var splitCharacter rune
	if kind == Kind_Synchronization {
		splitCharacter = '/'
	} else if kind == Kind_Forwarding {
		splitCharacter = ':'
	} else {
		panic("unhandled URL kind")
	}

	// Parse off the username. If we hit a '/', then we've reached the end of a
	// container specification and there was no username. Similarly, if we hit
	// the end of the string without seeing an '@', then there's also no
	// username specified. Ideally we'd want also to break on any character that
	// isn't allowed in a username, but that isn't well-defined, even for POSIX
	// (it's effectively determined by a configurable regular expression -
	// NAME_REGEX).
	var username string
	for i, r := range raw {
		if r == splitCharacter {
			break
		} else if r == '@' {
			username = raw[:i]
			raw = raw[i+1:]
			break
		}
	}

	// Split what remains into the container and the path (or forwarding
	// endpoint, depending on the URL kind). Ideally we'd want to be a bit more
	// stringent here about what characters we accept in container names,
	// potentially breaking early with an error if we see a "disallowed"
	// character, but we're better off just allowing Docker to reject container
	// names that it doesn't like.
	var container, path string
	for i, r := range raw {
		if r == splitCharacter {
			container = raw[:i]
			path = raw[i:]
			break
		}
	}
	if container == "" {
		return nil, errors.New("empty container name")
	} else if path == "" {
		if kind == Kind_Synchronization {
			return nil, errors.New("missing path")
		} else if kind == Kind_Forwarding {
			return nil, errors.New("missing forwarding endpoint")
		} else {
			panic("unhandled URL kind")
		}
	}

	// Perform path processing based on URL kind.
	if kind == Kind_Synchronization {
		// If the path starts with "/~", then we assume that it's supposed to be
		// a home-directory-relative path and remove the slash. At this point we
		// already know that the path starts with "/" since we retained that as
		// part of the path in the split operation above.
		if len(path) > 1 && path[1] == '~' {
			path = path[1:]
		}

		// If the path is of the form "/" + Windows path, then assume it's
		// supposed to be a Windows path. This is a heuristic, but a reasonable
		// one. We do this on all systems (not just on Windows as with SSH URLs)
		// because users can connect to Windows containers from non-Windows
		// systems. At this point we already know that the path starts with "/"
		// since we retained that as part of the path in the split operation
		// above.
		if isWindowsPath(path[1:]) {
			path = path[1:]
		}
	} else if kind == Kind_Forwarding {
		// For forwarding paths, we need to trim the split character at the
		// beginning.
		path = path[1:]

		// Parse the forwarding endpoint URL to ensure that it's valid.
		if _, _, err := forwarding.Parse(path); err != nil {
			return nil, errors.Wrap(err, "invalid forwarding endpoint URL")
		}
	} else {
		panic("unhandled URL kind")
	}

	// Store any Docker environment variables that we need to preserve. We only
	// store variables that are actually present, because Docker behavior will
	// vary depending on whether a variable is unset vs. set but empty.
	environment := make(map[string]string)
	for _, variable := range DockerEnvironmentVariables {
		if value, present := getEnvironmentVariable(variable, kind, first); present {
			environment[variable] = value
		}
	}

	// Success.
	return &URL{
		Kind:        kind,
		Protocol:    Protocol_Docker,
		User:        username,
		Host:        container,
		Path:        path,
		Environment: environment,
	}, nil
}
