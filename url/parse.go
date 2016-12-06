package url

import (
	"strconv"

	"github.com/pkg/errors"
)

type URL struct {
	Protocol Protocol
	Username string
	Hostname string
	Port     uint16
	Path     string
}

func Parse(raw string) (*URL, error) {
	switch classify(raw) {
	case ProtocolLocal:
		return &URL{
			Protocol: ProtocolLocal,
			Path:     raw,
		}, nil
	case ProtocolSSH:
		return parseSSH(raw)
	default:
		panic("unhandled URL protocol")
	}
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
	var port uint16
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
				port = uint16(port64)
				raw = raw[i+1:]
			}
		}

		// No need to continue scanning at this point.
		break
	}

	// Create the URL, using what remains as the path.
	return &URL{
		Protocol: ProtocolSSH,
		Username: username,
		Hostname: hostname,
		Port:     port,
		Path:     raw,
	}, nil
}
