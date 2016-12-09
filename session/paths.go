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
	replicasDirectoryName = "replicas"
	cachesDirectoryName   = "caches"
	stagingDirectoryName  = "staging"

	alphaSubdirectoryName = "alpha"
	betaSubdirectoryName  = "beta"
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

func pathForArchive(sessionIdentifier string) (string, error) {
	// Compute/create the archives directory.
	archivesDirectoryPath, err := filesystem.Mutagen(archivesDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create archives directory")
	}

	// Compute the combined path.
	return filepath.Join(archivesDirectoryPath, sessionIdentifier), nil
}

func pathForReplica(sessionIdentifier string, alpha bool) (string, error) {
	// Compute the endpoint subdirectory.
	endpointSubdirectoryName := alphaSubdirectoryName
	if !alpha {
		endpointSubdirectoryName = betaSubdirectoryName
	}

	// Compute/create the endpoint replicas directory.
	endpointReplicasDirectoryPath, err := filesystem.Mutagen(
		replicasDirectoryName,
		endpointSubdirectoryName,
	)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create endpoint replicas directory")
	}

	// Compute the combined path.
	return filepath.Join(endpointReplicasDirectoryPath, sessionIdentifier), nil
}

func pathForCache(sessionIdentifier string, alpha bool) (string, error) {
	// Compute the endpoint subdirectory.
	endpointSubdirectoryName := alphaSubdirectoryName
	if !alpha {
		endpointSubdirectoryName = betaSubdirectoryName
	}

	// Compute/create the endpoint caches directory.
	endpointCachesDirectoryPath, err := filesystem.Mutagen(
		cachesDirectoryName,
		endpointSubdirectoryName,
	)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute/create endpoint caches directory")
	}

	// Compute the combined path.
	return filepath.Join(endpointCachesDirectoryPath, sessionIdentifier), nil
}

func pathForStaging(sessionIdentifier string, alpha bool) (string, error) {
	// Compute the endpoint subdirectory.
	endpointSubdirectoryName := alphaSubdirectoryName
	if !alpha {
		endpointSubdirectoryName = betaSubdirectoryName
	}

	// Compute/create the endpoint staging directory.
	return filesystem.Mutagen(
		stagingDirectoryName,
		endpointSubdirectoryName,
		sessionIdentifier,
	)
}

func nameForStaging(path string, digest []byte) string {
	return fmt.Sprintf("%x-%x", sha1.Sum([]byte(path)), digest)
}
