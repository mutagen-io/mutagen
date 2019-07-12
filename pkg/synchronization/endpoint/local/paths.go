package local

import (
	"crypto/sha1"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

const (
	// alphaName is the name to use for alpha when distinguishing endpoints.
	alphaName = "alpha"
	// betaName is the name to use for beta when distinguishing endpoints.
	betaName = "beta"

	// stagingPrefixLength is the byte length to use for prefix directories when
	// load-balancing staged files.
	stagingPrefixLength = 1
)

// pathForCache computes the path to the serialized cache for the given session
// identifier and endpoint role.
func pathForCache(session string, alpha bool) (string, error) {
	// Compute/create the caches directory.
	cachesDirectoryPath, err := filesystem.Mutagen(true, filesystem.MutagenCachesDirectoryName)
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

// pathForMutagenStagingRoot computes the path to the staging root in the
// Mutagen data directory for the given session identifier and endpoint. It
// ensures that staging subdirectory of the Mutagen data directory exists, but
// it does not create the staging root itself.
func pathForMutagenStagingRoot(session string, alpha bool) (string, error) {
	// Compute the path to the staging root parent (the global Mutagen data
	// directory in which staging roots are stored) and ensure that it exists.
	stagingDataPath, err := filesystem.Mutagen(true, filesystem.MutagenStagingDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to create staging data directory")
	}

	// Compute the endpoint name.
	endpointName := alphaName
	if !alpha {
		endpointName = betaName
	}

	// Compute the staging root name.
	stagingRootName := fmt.Sprintf("%s-%s", session, endpointName)

	// Compute the combined path.
	return filepath.Join(stagingDataPath, stagingRootName), nil
}

// pathForNeighboringStagingRoot computes the path to the staging root which
// neighbors the synchronization root for the given root, session identifier,
// and endpoint. It does not create the directory or any parent directories.
func pathForNeighboringStagingRoot(root, session string, alpha bool) (string, error) {
	// Compute the parent of the staging root.
	parent := filepath.Dir(root)

	// Compute the endpoint name.
	endpointName := alphaName
	if !alpha {
		endpointName = betaName
	}

	// Compute the name of the staging directory.
	stagingRootName := fmt.Sprintf(
		"%sstaging-%s-%s",
		filesystem.TemporaryNamePrefix,
		session,
		endpointName,
	)

	// Compute the path to the staging root.
	return filepath.Join(parent, stagingRootName), nil
}

// pathForStaging computes the staging path for the specified path/digest
// relative to the staging root. It returns the prefix directory name but does
// not ensure that it's been created.
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
