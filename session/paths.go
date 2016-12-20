package session

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
)

const (
	sessionsDirectoryName = "sessions"
	archivesDirectoryName = "archives"
)

// TODO: Note that an empty session identifier will return the sessions
// directory path.
func pathForSession(sessionIdentifier string) (string, error) {
	// Compute/create the sessions directory.
	sessionsDirectoryPath, err := filesystem.Mutagen(sessionsDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create sessions directory")
	}

	// Success.
	return filepath.Join(sessionsDirectoryPath, sessionIdentifier), nil
}

func pathForArchive(session string) (string, error) {
	// Compute/create the archives directory.
	archivesDirectoryPath, err := filesystem.Mutagen(archivesDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create archives directory")
	}

	// Success.
	return filepath.Join(archivesDirectoryPath, session), nil
}
