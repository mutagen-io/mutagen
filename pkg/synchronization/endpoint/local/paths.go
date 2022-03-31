package local

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
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
	cachesDirectoryPath, err := filesystem.Mutagen(true, filesystem.MutagenSynchronizationCachesDirectoryName)
	if err != nil {
		return "", fmt.Errorf("unable to compute/create caches directory: %w", err)
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
	stagingDataPath, err := filesystem.Mutagen(true, filesystem.MutagenSynchronizationStagingDirectoryName)
	if err != nil {
		return "", fmt.Errorf("unable to create staging data directory: %w", err)
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

// pathForInternalStagingRoot computes the path to the staging root which is
// internal to the synchronization root for the given root, session identifier,
// and endpoint. It does not create the directory or any parent directories.
func pathForInternalStagingRoot(root, session string, alpha bool) (string, error) {
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
	return filepath.Join(root, stagingRootName), nil
}

// pathForStaging computes the staging path for the specified path/digest
// relative to the staging root. It also returns the prefix directory byte value
// and name, though it does not create the prefix directory.
func pathForStaging(root, path string, digest []byte) (string, byte, string, error) {
	// Ensure that the digest is non-empty. We need at least one byte for the
	// staging prefix to be valid, but beyond that we don't know what digest
	// length is in-use (or might be in use in the future).
	if len(digest) == 0 {
		return "", 0, "", errors.New("entry digest too short")
	}
	prefixByte := digest[0]

	// Convert the digest to hexadecimal encoding and extract the prefix.
	digestHex := hex.EncodeToString(digest)
	prefix := digestHex[:2]

	// Compute the hexadecimal encoded digest of the path name.
	pathDigest := sha1.Sum([]byte(path))
	pathDigestHex := hex.EncodeToString(pathDigest[:])

	// Compute the staging name.
	stagingName := pathDigestHex + "_" + digestHex

	// Success.
	return filepath.Join(root, prefix, stagingName), prefixByte, prefix, nil
}
