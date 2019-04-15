package url

import (
	"github.com/pkg/errors"
)

// Parse parses a raw URL string into a URL type.
func Parse(raw string, alpha bool) (*URL, error) {
	// Don't allow empty raw URLs.
	if raw == "" {
		return nil, errors.New("empty URL")
	}

	// Dispatch URL parsing based on type. We have to be careful about the
	// ordering here because URLs may be classified as multiple types (e.g. a
	// Docker URL would also be classified as an SCP-style SSH URL), but we only
	// want them to be parsed according to the better and more specific match.
	// If we don't match anything, we assume the URL is a local path.
	if isDockerURL(raw) {
		return parseDocker(raw, alpha)
	} else if isKubectlURL(raw) {
		return parseKubectl(raw, alpha)
	} else if isSCPSSHURL(raw) {
		return parseSCPSSH(raw)
	} else {
		return parseLocal(raw)
	}
}
