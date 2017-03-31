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
	// Parse off the username. If we hit a colon, then we've reached the end of
	// a hostname specification and there was no username. Ideally we'd want to
	// break on any character that isn't allowed in a username, but that isn't
	// well-defined, even for POSIX (it's effectively determined by a
	// configurable regular expression - NAME_REGEX).
	var username string
	for i, r := range raw {
		if r == ':' {
			break
		} else if r == '@' {
			username = raw[:i]
			raw = raw[i+1:]
			break
		}
	}

	// Parse off the host. Again, ideally we'd want to be a bit more stringent
	// here about what characters we accept in hostnames, potentially breaking
	// early with an error if we see a "disallowed" character, but we're better
	// off just allowing SSH to reject hostnames that it doesn't like, because
	// with its aliases it's hard to say what it'll allow.
	var hostname string
	for i, r := range raw {
		if r == ':' {
			hostname = raw[:i]
			raw = raw[i+1:]
			break
		}
	}
	if hostname == "" {
		return nil, errors.New("invalid hostname")
	}

	// Parse off the port. This is not a standard SCP URL syntax (and even Git
	// makes you use full SSH URLs if you want to specify a port), so we invent
	// our own rules here, but essentially we just scan until the next colon,
	// and if there is one, and all characters before it are 0-9, we try to
	// parse them as a port (restricting to the allowed port range). On failure,
	// we just treat it as part of the path. We only accept non-empty strings,
	// because technically a file could start with ':' on some systems. In the
	// rare case that a path begins with something like "#:" (where # is a
	// numeric sequence that could be mistaken for a port), an absolute path can
	// be specified.
	var port uint32
	for i, r := range raw {
		// If we're in a string of characters, keep going.
		if '0' <= r && r <= '9' {
			continue
		}

		// If we've encountered a colon and we're not at the beginning of the
		// remaining string, attempt to parse the preceeding value as a port.
		if r == ':' && i != 0 {
			if port64, err := strconv.ParseUint(raw[:i], 10, 16); err == nil {
				port = uint32(port64)
				raw = raw[i+1:]
			}
		}

		// No need to continue scanning at this point. Either we successfully
		// parsed, failed to parse, or hit a character that wasn't numeric.
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
