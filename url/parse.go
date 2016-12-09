package url

import (
	"strconv"

	"github.com/pkg/errors"
)

func Parse(raw string) (*URL, error) {
	// Don't allow empty raw URLs.
	if raw == "" {
		return nil, errors.New("empty raw URL")
	}

	// Check if this is an SCP-style URL. A URL is classified as such if it
	// contains a colon with no forward slashes before it. On Windows, paths
	// beginning with x:\ or x:/ (where x is a-z or A-Z) are almost certainly
	// referring to local paths, but will trigger the SCP URL detection, so we
	// have to check those first. This is, of course, something of a heuristic,
	// but we're unlikely to encounter 1-character hostnames and very likely to
	// encounter Windows paths (except on POSIX, where this check always returns
	// false). If Windows users do have a 1-character hostname, they should just
	// use some other addressing scheme for it (e.g. an IP address or alternate
	// hostname).
	if !isWindowsPath(raw) {
		for _, c := range raw {
			if c == ':' {
				return parseSSH(raw)
			} else if c == '/' {
				break
			}
		}
	}

	// Otherwise, just treat this as a raw path.
	return &URL{
		Protocol: Protocol_Local,
		Path:     raw,
	}, nil
}

func parseSSH(raw string) (*URL, error) {
	// Parse off the username.
	var username string
	for i, r := range raw {
		if r == ':' {
			break
		} else if r == '@' {
			username = raw[:i]
			raw = raw[i+1:]
		}
	}

	// Parse off the host.
	var hostname string
	for i, r := range raw {
		if r == ':' {
			hostname = raw[:i]
			raw = raw[i+1:]
		}
	}
	if hostname == "" {
		return nil, errors.New("invalid hostname")
	}

	// Parse off the port. This is not a standard SCP URL syntax (and even Git
	// makes you use full SSH URLs if you want to specify a port), so we invent
	// our own rules here, but essentially we just scan until the next colon,
	// and if there is one and all characters before it are 0-9, we try to parse
	// them as a port. We only accept non-empty strings, because technically a
	// file could start with ':' on some systems.
	var port uint32
	for i, r := range raw {
		// If we're in a string of characters, keep going.
		if '0' <= r && r <= '9' {
			continue
		}

		// If we've encountered a colon and we're not at the beginning of the
		// remainder, attempt to parse the preceeding value as a port.
		if r == ':' && i != 0 {
			if port64, err := strconv.ParseUint(raw[:i], 10, 16); err != nil {
				// If parsing fails, then just assume that the user wasn't
				// attempting to specify a port.
				break
			} else {
				port = uint32(port64)
				raw = raw[i+1:]
			}
		}

		// No need to continue scanning at this point.
		break
	}

	// Create the URL, using what remains as the path.
	return &URL{
		Protocol: Protocol_SSH,
		Username: username,
		Hostname: hostname,
		Port:     port,
		Path:     raw,
	}, nil
}
