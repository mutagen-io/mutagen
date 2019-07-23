package forwarding

import (
	"strings"

	"github.com/pkg/errors"
)

// Parse parses a forwarding sub-URL (which is stored as the Path component of
// an endpoint URL) into protocol and address components.
func Parse(url string) (string, string, error) {
	// Ensure that the URL is non-empty.
	if url == "" {
		return "", "", errors.New("empty URL")
	}

	// Split the specification and ensure that it was correctly formatted.
	components := strings.SplitN(url, ":", 2)
	if len(components) != 2 {
		return "", "", errors.New("incorrectly formatted URL")
	}

	// Ensure that the protocol is valid.
	if !IsValidProtocol(components[0]) {
		return "", "", errors.Errorf("invalid protocol: %s", components[0])
	}

	// Ensure that the address is non-empty. There's not much other validation
	// that we can do easily.
	if components[1] == "" {
		return "", "", errors.New("empty address")
	}

	// Success.
	return components[0], components[1], nil
}
