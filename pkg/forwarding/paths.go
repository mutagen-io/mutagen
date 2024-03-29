package forwarding

import (
	"fmt"
	"path/filepath"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

const (
	// sessionsDirectoryName is the name of the session storage directory within
	// the forwarding data directory.
	sessionsDirectoryName = "sessions"
)

// pathForSession computes the path to the serialized session for the given
// session identifier. An empty session identifier will return the sessions
// directory path.
func pathForSession(sessionIdentifier string) (string, error) {
	// Compute/create the sessions directory.
	sessionsDirectoryPath, err := filesystem.Mutagen(
		true,
		filesystem.MutagenForwardingDirectoryName,
		sessionsDirectoryName,
	)
	if err != nil {
		return "", fmt.Errorf("unable to compute/create sessions directory: %w", err)
	}

	// Success.
	return filepath.Join(sessionsDirectoryPath, sessionIdentifier), nil
}
