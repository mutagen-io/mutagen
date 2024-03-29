package url

import (
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// parseLocal parses a local URL. It simply assumes the URL refers to a local
// path or forwarding endpoint specification.
func parseLocal(raw string, kind Kind) (*URL, error) {
	// If this is a synchronization URL, then ensure that its path is
	// normalized.
	if kind == Kind_Synchronization {
		if normalized, err := filesystem.Normalize(raw); err != nil {
			return nil, fmt.Errorf("unable to normalize path: %w", err)
		} else {
			raw = normalized
		}
	}

	// If this is a forwarding URL, then parse it to ensure that it's valid. If
	// it's a Unix domain socket endpoint, then ensure that the socket path is
	// normalized.
	if kind == Kind_Forwarding {
		// Perform parsing.
		protocol, address, err := forwarding.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid forwarding endpoint URL: %w", err)
		}

		// Normalize and reformat the endpoint URL if necessary.
		if protocol == "unix" {
			if normalized, err := filesystem.Normalize(address); err != nil {
				return nil, fmt.Errorf("unable to normalize socket path: %w", err)
			} else {
				raw = protocol + ":" + normalized
			}
		}
	}

	// Create the URL.
	return &URL{
		Kind:     kind,
		Protocol: Protocol_Local,
		Path:     raw,
	}, nil
}
