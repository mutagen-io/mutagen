package filesystem

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	// MutagenDataDirectoryName is the name of the global Mutagen data directory
	// inside the user's home directory.
	MutagenDataDirectoryName = ".mutagen"

	// MutagenConfigurationName is the name of the global Mutagen configuration
	// file inside the user's home directory.
	MutagenConfigurationName = ".mutagen.toml"

	// MutagenDaemonDirectoryName is the name of the daemon storage directory
	// within the Mutagen data directory.
	MutagenDaemonDirectoryName = "daemon"

	// MutagenAgentsDirectoryName is the name of the agent storage directory
	// within the Mutagen data directory.
	MutagenAgentsDirectoryName = "agents"

	// MutagenSessionsDirectoryName is the name of the session storage directory
	// within the Mutagen data directory.
	MutagenSessionsDirectoryName = "sessions"

	// MutagenCachesDirectoryName is the name of the cache storage directory
	// within the Mutagen data directory.
	MutagenCachesDirectoryName = "caches"

	// MutagenArchivesDirectoryName is the name of the archive storage directory
	// within the Mutagen data directory.
	MutagenArchivesDirectoryName = "archives"

	// MutagenStagingDirectoryName is the name of the staging storage directory
	// within the Mutagen data directory.
	MutagenStagingDirectoryName = "staging"
)

// Mutagen computes (and optionally creates) subdirectories inside the Mutagen
// data directory.
func Mutagen(create bool, pathComponents ...string) (string, error) {
	// Compute the path to the user's home directory.
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "unable to compute path to home directory")
	}

	// Compute the path to the Mutagen data directory.
	mutagenDataDirectoryPath := filepath.Join(homeDirectory, MutagenDataDirectoryName)

	// Compute the target path.
	result := filepath.Join(mutagenDataDirectoryPath, filepath.Join(pathComponents...))

	// If requested, attempt to create the Mutagen directory and the specified
	// subpath. Also ensure that the Mutagen data directory is hidden.
	// TODO: Should we iterate through each component and ensure the user hasn't
	// changed the directory permissions? MkdirAll won't reset them. But I
	// suppose the user may have changed them for whatever reason (though I
	// can't imagine any).
	if create {
		if err := os.MkdirAll(result, 0700); err != nil {
			return "", errors.Wrap(err, "unable to create subpath")
		} else if err := MarkHidden(mutagenDataDirectoryPath); err != nil {
			return "", errors.Wrap(err, "unable to hide Mutagen data directory")
		}
	}

	// Success.
	return result, nil
}
