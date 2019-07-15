package url

import (
	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// parseLocal parses a local URL. It simply assumes the URL refers to a local
// path or forwarding endpoint specification.
func parseLocal(raw string, kind Kind) (*URL, error) {
	// If this is a forwarding URL, then parse it to ensure that it's valid.
	if kind == Kind_Forwarding {
		if _, _, err := forwarding.Parse(raw); err != nil {
			return nil, errors.Wrap(err, "invalid forwarding endpoint URL")
		}
	}

	// Create the URL.
	return &URL{
		Kind:     kind,
		Protocol: Protocol_Local,
		Path:     raw,
	}, nil
}
