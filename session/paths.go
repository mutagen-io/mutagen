package session

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
)

const (
	sessionsDirectoryName = "sessions"
)

// TODO: Note that an empty session identifier will return the sessions
// directory path.
func pathForSession(sessionIdentifier string) (string, error) {
	// Compute/create the sessions directory.
	sessionsDirectoryPath, err := filesystem.Mutagen(sessionsDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create sessions directory")
	}

	// Compute the combined path.
	return filepath.Join(sessionsDirectoryPath, sessionIdentifier), nil
}
