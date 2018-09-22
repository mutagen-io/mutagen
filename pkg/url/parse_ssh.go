package url

import (
	"runtime"
	"strconv"

	"github.com/pkg/errors"
)

// isSCPSSHURL determines whether or not a raw URL is an SCP-style SSH URL. A
// URL is classified as such if it contains a colon with no forward slashes
// before it. On Windows, paths beginning with x:\ or x:/ (where x is a-z or
// A-Z) are almost certainly referring to local paths, but will trigger the SCP
// URL detection, so we have to check those first. This is, of course, something
// of a heuristic, but we're unlikely to encounter 1-character hostnames and
// very likely to encounter Windows paths (except on POSIX, where this check
// always returns false). If Windows users do have a 1-character hostname, they
// should just use some other addressing scheme for it (e.g. an IP address or
// alternate hostname).
func isSCPSSHURL(raw string) bool {
	// If this is a Windows path on a Windows system, then reject it because it
	// should be treated as a local URL.
	if runtime.GOOS == "windows" && isWindowsPath(raw) {
		return false
	}

	// Otherwise check if there's a colon that comes before all forward slashes.
	for _, c := range raw {
		if c == ':' {
			return true
		} else if c == '/' {
			break
		}
	}

	// If there wasn't a colon or a slash came first, then this is not an
	// SCP-style SSH URL.
	return false
}

// parseSCPSSH parses an SCP-style SSH URL.
func parseSCPSSH(raw string) (*URL, error) {
	// Parse off the username. If we hit a ':', then we've reached the end of
	// the hostname specification and there was no username. Similarly, if we
	// hit the end of the string without seeing an '@', then there's also no
	// username specified. Ideally we'd want to break on any character that
	// isn't allowed in a username, but that isn't well-defined, even for POSIX
	// (it's effectively determined by a configurable regular expression -
	// NAME_REGEX). We enforce that if a username is specified, that it is
	// non-empty.
	var username string
	for i, r := range raw {
		if r == ':' {
			break
		} else if r == '@' {
			if i == 0 {
				return nil, errors.New("empty username specified")
			}
			username = raw[:i]
			raw = raw[i+1:]
			break
		}
	}

	// Parse off the host. Again, ideally we'd want to be a bit more stringent
	// here about what characters we accept in hostnames, potentially breaking
	// early with an error if we see a "disallowed" character, but we're better
	// off just allowing SSH to reject hostnames that it doesn't like, because
	// with its aliases it's hard to say what it'll allow. We reject empty
	// hostnames and we reject cases where we've scanned the entire string and
	// not found a colon (which indicates that this is probably not an SCP-style
	// SSH URL).
	var hostname string
	for i, r := range raw {
		if r == ':' {
			if i == 0 {
				return nil, errors.New("empty hostname")
			}
			hostname = raw[:i]
			raw = raw[i+1:]
			break
		}
	}
	if hostname == "" {
		return nil, errors.New("no hostname present")
	}

	// Parse off the port. This is not a standard SCP URL syntax (and even Git
	// makes you use full SSH URLs if you want to specify a port), so we invent
	// our own rules here, but essentially we just scan until the next colon,
	// and if there is one, and all characters before it are 0-9, we try to
	// parse the preceding segment as a port (restricting to the allowed port
	// range). We allow such digit strings to be empty, because that probably
	// indicates an attempt to specify a port. In the rare case that a path
	// begins with something like "#:" (where # is a (potentially empty) digit
	// sequence that could be mistaken for a port), an absolute or home-relative
	// path can be specified.
	var port uint32
	for i, r := range raw {
		// If we're in a string of digits, keep going.
		if '0' <= r && r <= '9' {
			continue
		}

		// If we've encountered a colon, then attempt to parse the preceding
		// portion of the string as a port value.
		if r == ':' {
			if port64, err := strconv.ParseUint(raw[:i], 10, 16); err != nil {
				return nil, errors.New("invalid port value specified")
			} else {
				port = uint32(port64)
				raw = raw[i+1:]
			}
		}

		// No need to continue scanning at this point. Either we successfully
		// parsed, failed to parse, or hit a character that wasn't numeric.
		break
	}

	// Treat what remains as the path. Ensure that it's non-empty.
	path := raw
	if path == "" {
		return nil, errors.New("empty path")
	}

	// Create the URL, using what remains as the path.
	return &URL{
		Protocol: Protocol_SSH,
		Username: username,
		Hostname: hostname,
		Port:     port,
		Path:     path,
	}, nil
}
