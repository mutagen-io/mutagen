package session

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

const (
	// sessionsDirectoryName is the name of the sessions subdirectory within the
	// Mutagen directory.
	sessionsDirectoryName = "sessions"
	// archivesDirectoryName is the name of the archives subdirectory within the
	// Mutagen directory.
	archivesDirectoryName = "archives"
)

// pathForSession computes the path to the serialized session for the given
// session identifier. An empty session identifier will return the sessions
// directory path.
func pathForSession(sessionIdentifier string) (string, error) {
	// Compute/create the sessions directory.
	sessionsDirectoryPath, err := filesystem.Mutagen(true, sessionsDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create sessions directory")
	}

	// Success.
	return filepath.Join(sessionsDirectoryPath, sessionIdentifier), nil
}

// pathForArchive computes the path to the serialized archive for the given
// session identifier.
func pathForArchive(session string) (string, error) {
	// Compute/create the archives directory.
	archivesDirectoryPath, err := filesystem.Mutagen(true, archivesDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create archives directory")
	}

	// Success.
	return filepath.Join(archivesDirectoryPath, session), nil
}
