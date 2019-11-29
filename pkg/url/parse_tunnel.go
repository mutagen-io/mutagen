package url

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

const (
	// tunnelURLPrefix is the lowercase version of the tunnel URL prefix.
	tunnelURLPrefix = "tunnel://"
)

// isTunnelURL checks whether or not a URL is a tunnel URL. It requires the
// presence of a tunnel protocol prefix.
func isTunnelURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(raw), tunnelURLPrefix)
}

// parseTunnel parses a Docker URL.
func parseTunnel(raw string, kind Kind) (*URL, error) {
	// Strip off the prefix.
	raw = raw[len(tunnelURLPrefix):]

	// Determine the character that splits the tunnel identifier/name from the
	// path or forwarding endpoint component.
	var splitCharacter rune
	if kind == Kind_Synchronization {
		splitCharacter = '/'
	} else if kind == Kind_Forwarding {
		splitCharacter = ':'
	} else {
		panic("unhandled URL kind")
	}

	// Ensure that no username is specified.
	for _, r := range raw {
		if r == splitCharacter {
			break
		} else if r == '@' {
			return nil, errors.New("user specification not allowed for tunnel URLs")
		}
	}

	// Split what remains into the tunnel identifier/name and the path (or
	// forwarding endpoint, depending on the URL kind). Ideally we'd want to be
	// a bit more stringent here about what characters we accept in tunnel
	// identifiers/names, potentially breaking early with an error if we see a
	// "disallowed" character, but we're better off just allowing the tunnel
	// manager to reject tunnel identifiers and names that it doesn't like.
	var identifierOrName, path string
	for i, r := range raw {
		if r == splitCharacter {
			identifierOrName = raw[:i]
			path = raw[i:]
			break
		}
	}
	if identifierOrName == "" {
		return nil, errors.New("empty tunnel identifier/name")
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
		// because users can connect to Windows systems from non-Windows
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

	// Success.
	return &URL{
		Kind:     kind,
		Protocol: Protocol_Tunnel,
		Host:     identifierOrName,
		Path:     path,
	}, nil
}
