package session

import (
	"crypto/sha1"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/filesystem"
	"github.com/havoc-io/mutagen/sync"
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

func pathForCache(session string, alpha bool) (string, error) {
	// Compute/create the caches directory.
	cachesDirectoryPath, err := filesystem.Mutagen(cachesDirectoryName)
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

	// Compute/create the staging root.
	return filesystem.Mutagen(stagingDirectoryName, stagingRootName)
}

func pathForStaging(session string, alpha bool, path string, entry *sync.Entry) (string, error) {
	// Compute the endpoint name.
	endpointName := alphaName
	if !alpha {
		endpointName = betaName
	}

	// Compute the staging root name.
	stagingRootName := fmt.Sprintf("%s_%s", session, endpointName)

	// Compute the prefix for the entry.
	if len(entry.Digest) < (stagingPrefixLength + 1) {
		return "", errors.New("entry digest too short")
	}
	prefixBytes := entry.Digest[:stagingPrefixLength]
	prefix := fmt.Sprintf("%x", prefixBytes)

	// Compute/create the staging prefix directory.
	prefixDirectoryPath, err := filesystem.Mutagen(stagingDirectoryName, stagingRootName, prefix)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create staging prefix directory")
	}

	// Compute the staging name.
	stagingName := fmt.Sprintf("%x_%t_%x",
		sha1.Sum([]byte(path)), entry.Executable, entry.Digest,
	)

	// Success.
	return filepath.Join(prefixDirectoryPath, stagingName), nil
}
