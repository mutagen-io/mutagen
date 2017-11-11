package session

import (
	"crypto/sha1"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
)

const (
	sessionsDirectoryName = "sessions"
	archivesDirectoryName = "archives"
	cachesDirectoryName   = "caches"
	stagingDirectoryName  = "staging"

	alphaName = "alpha"
	betaName  = "beta"

	stagingPrefixLength = 1
)

// TODO: Note that an empty session identifier will return the sessions
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

func pathForArchive(session string) (string, error) {
	// Compute/create the archives directory.
	archivesDirectoryPath, err := filesystem.Mutagen(true, archivesDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create archives directory")
	}

	// Success.
	return filepath.Join(archivesDirectoryPath, session), nil
}

func pathForCache(session string, alpha bool) (string, error) {
	// Compute/create the caches directory.
	cachesDirectoryPath, err := filesystem.Mutagen(true, cachesDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create caches directory")
	}

	// Compute the endpoint name.
	endpointName := alphaName
	if !alpha {
		endpointName = betaName
	}

	// Compute the cache name.
	cacheName := fmt.Sprintf("%s_%s", session, endpointName)

	// Success.
	return filepath.Join(cachesDirectoryPath, cacheName), nil
}

func pathForStagingRoot(session string, alpha bool) (string, error) {
	// Compute the endpoint name.
	endpointName := alphaName
	if !alpha {
		endpointName = betaName
	}

	// Compute the staging root name.
	stagingRootName := fmt.Sprintf("%s_%s", session, endpointName)

	// Compute the staging root, but don't create it.
	return filesystem.Mutagen(false, stagingDirectoryName, stagingRootName)
}

func pathForStaging(root, path string, digest []byte) (string, string, error) {
	// Compute the prefix for the entry.
	if len(digest) == 0 {
		return "", "", errors.New("entry digest too short")
	}
	prefix := fmt.Sprintf("%x", digest[:1])

	// Compute the staging name.
	stagingName := fmt.Sprintf("%x_%x", sha1.Sum([]byte(path)), digest)

	// Success.
	return filepath.Join(root, prefix, stagingName), prefix, nil
}
