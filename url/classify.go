package url

type Protocol uint8

const (
	ProtocolLocal Protocol = iota
	ProtocolSSH
)

func classify(raw string) Protocol {
	// On Windows, paths beginning with x:\ or x:/ (where x is a-z or A-Z) are
	// almost certainly referring to local paths, but will trigger the SCP URL
	// detection, so we have to check those first. This is, of course, something
	// of a heuristic, but we're unlikely to encounter 1-character hostnames and
	// very likely to encounter Windows paths (except on POSIX, where this check
	// always returns false). If Windows users do have a 1-character hostname,
	// they should just use some other addressing scheme for it (e.g. an IP
	// address or alternate hostname).
	if isWindowsPath(raw) {
		return ProtocolLocal
	}

	// Check if this is an SCP-style URL. A URL is classified as such if it
	// contains a colon with no forward slashes before it.
	for _, c := range raw {
		if c == ':' {
			return ProtocolSSH
		} else if c == '/' {
			break
		}
	}

	// Otherwise, assume this is a raw path.
	return ProtocolLocal
}
