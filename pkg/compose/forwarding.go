package compose

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mutagen-io/mutagen/pkg/docker"
	"github.com/mutagen-io/mutagen/pkg/url"
	forwardingurl "github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// networkURLPrefix is the lowercase version of the network URL prefix.
const networkURLPrefix = "network://"

// isNetworkURL checks if raw URL is a Docker Compose network pseudo-URL.
func isNetworkURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(raw), networkURLPrefix)
}

// isTCPForwardingProtocol checks if a forwarding protocol is TCP-based.
func isTCPForwardingProtocol(protocol string) bool {
	switch protocol {
	case "tcp":
		return true
	case "tcp4":
		return true
	case "tcp6":
		return true
	default:
		return false
	}
}

// parseNetworkURL parses a Docker Compose network pseudo-URL, converting it to
// a concrete Mutagen Docker forwarding URL and enforcing that its forwarding
// endpoint protocol is TCP-based. It looks for Docker environment variables in
// the specified project environment and uses the specified daemon connection
// flags to determine URL parameters. This function also returns the network
// dependency for the URL. This function must only be called on URLs that have
// been classified as network URLs by isNetworkURL, otherwise it may panic.
func parseNetworkURL(
	raw, mutagenContainerName string,
	environment map[string]string,
	daemonFlags docker.DaemonConnectionFlags,
) (*url.URL, string, error) {
	// Strip off the prefix
	raw = raw[len(networkURLPrefix):]

	// Find the first colon, which will indicate the end of the network name.
	var network, endpoint string
	if colonIndex := strings.IndexByte(raw, ':'); colonIndex < 0 {
		return nil, "", errors.New("unable to find forwarding endpoint specification")
	} else if colonIndex == 0 {
		return nil, "", errors.New("empty network name")
	} else {
		network = raw[:colonIndex]
		endpoint = raw[colonIndex+1:]
	}

	// Parse the forwarding endpoint URL to ensure that it's valid and supported
	// for use with Docker Compose.
	if protocol, _, err := forwardingurl.Parse(endpoint); err != nil {
		return nil, "", fmt.Errorf("invalid forwarding endpoint URL: %w", err)
	} else if !isTCPForwardingProtocol(protocol) {
		return nil, "", fmt.Errorf("non-TCP-based forwarding endpoint (%s) unsupported", endpoint)
	}

	// Store any Docker environment variables that we need to preserve. We only
	// store variables that are actually present, because Docker behavior will
	// vary depending on whether a variable is unset vs. set but empty. Note
	// that unlike standard Docker URL parsing, we load these variables from the
	// project environment (which may include values from "dotenv" files). We
	// also don't support endpoint-specific variants since those don't make
	// sense in the context of Docker Compose.
	urlEnvironment := make(map[string]string)
	for _, variable := range url.DockerEnvironmentVariables {
		if value, present := environment[variable]; present {
			urlEnvironment[variable] = value
		}
	}

	// Create a Docker forwarding URL.
	return &url.URL{
		Kind:        url.Kind_Forwarding,
		Protocol:    url.Protocol_Docker,
		Host:        mutagenContainerName,
		Path:        endpoint,
		Environment: urlEnvironment,
		Parameters:  daemonFlags.ToURLParameters(),
	}, network, nil
}
