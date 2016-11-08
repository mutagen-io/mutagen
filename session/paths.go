package session

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
)

const (
	sessionsDirectoryName = "sessions"
	stagingDirectoryName  = "staging"
)

func path(session string) (string, error) {
	return filesystem.Mutagen(sessionsDirectoryName, session)
}

func subpath(session, subpath string) (string, error) {
	// Compute the session root.
	root, err := path(session)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute session directory")
	}

	// Compute the combined path.
	return filepath.Join(root, subpath), nil
}

func stagingPath(session string) (string, error) {
	return filesystem.Mutagen(sessionsDirectoryName, session, stagingDirectoryName)
}
