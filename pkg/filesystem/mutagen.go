package filesystem

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem/locking"
)

const (
	// MutagenLockFileName is the name of the lock file coordinating access to
	// the Mutagen data directory.
	MutagenLockFileName = ".mutagen.lock"

	// MutagenDataDirectoryName is the name of the Mutagen data directory.
	MutagenDataDirectoryName = ".mutagen"

	// mutagenConfigurationName is the name of the global Mutagen configuration
	// file inside the user's home directory.
	mutagenConfigurationName = ".mutagen.toml"

	// MutagenDaemonDirectoryName is the name of the daemon subdirectory within
	// the Mutagen data directory.
	MutagenDaemonDirectoryName = "daemon"

	// MutagenAgentsDirectoryName is the subdirectory of the Mutagen directory
	// in which agents should be stored.
	MutagenAgentsDirectoryName = "agents"

	// MutagenSessionsDirectoryName is the name of the sessions subdirectory
	// within the Mutagen data directory.
	MutagenSessionsDirectoryName = "sessions"

	// MutagenCachesDirectoryName is the name of the caches subdirectory within
	// the Mutagen data directory.
	MutagenCachesDirectoryName = "caches"

	// MutagenArchivesDirectoryName is the name of the archives subdirectory
	// within the Mutagen data directory.
	MutagenArchivesDirectoryName = "archives"

	// MutagenStagingDirectoryName is the name of the staging subdirectory
	// within the Mutagen data directory.
	MutagenStagingDirectoryName = "staging"
)

// HomeDirectory is the cached path to the current user's home directory.
var HomeDirectory string

// MutagenLockFilePath is the path to the lock file coordinating access to the
// Mutagen data directory. It can be overridden in init functions or entry
// points, but this should be done before any calls to AcquireMutagenLock.
var MutagenLockFilePath string

// MutagenDataDirectoryPath is the path to the Mutagen data directory. It can be
// overridden in init functions or entry points, but this should be done before
// any calls to Mutagen.
var MutagenDataDirectoryPath string

// MutagenConfigurationPath is the path to the global Mutagen configuration
// file.
var MutagenConfigurationPath string

// init performs global initialization.
func init() {
	// Grab the current user's home directory.
	if h, err := os.UserHomeDir(); err != nil {
		panic(errors.Wrap(err, "unable to query user's home directory"))
	} else if h == "" {
		panic(errors.New("home directory path empty"))
	} else {
		HomeDirectory = h
	}

	// Compute the path to the Mutagen lock file.
	MutagenLockFilePath = filepath.Join(HomeDirectory, MutagenLockFileName)

	// Compute the path to the Mutagen data directory.
	MutagenDataDirectoryPath = filepath.Join(HomeDirectory, MutagenDataDirectoryName)

	// Compute the path to the configuration file.
	MutagenConfigurationPath = filepath.Join(HomeDirectory, mutagenConfigurationName)
}

// AcquireMutagenLock is a convenience function which attempts to acquire the
// Mutagen data directory lock and returns a locked file locker.
func AcquireMutagenLock() (*locking.Locker, error) {
	// Create the locker and attempt to acquire the lock.
	locker, err := locking.NewLocker(MutagenLockFilePath, 0600)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create file locker")
	} else if err = locker.Lock(false); err != nil {
		locker.Close()
		return nil, err
	}

	// Success.
	return locker, nil
}

// Mutagen computes (and optionally creates) subdirectories inside the Mutagen
// data directory.
func Mutagen(create bool, pathComponents ...string) (string, error) {
	// Compute the target path.
	result := filepath.Join(MutagenDataDirectoryPath, filepath.Join(pathComponents...))

	// If requested, attempt to create the Mutagen directory and the specified
	// subpath. Also ensure that the Mutagen data directory is hidden.
	// TODO: Should we iterate through each component and ensure the user hasn't
	// changed the directory permissions? MkdirAll won't reset them. But I
	// suppose the user may have changed them for whatever reason (though I
	// can't imagine any).
	if create {
		if err := os.MkdirAll(result, 0700); err != nil {
			return "", errors.Wrap(err, "unable to create subpath")
		} else if err := MarkHidden(MutagenDataDirectoryPath); err != nil {
			return "", errors.Wrap(err, "unable to hide Mutagen data directory")
		}
	}

	// Success.
	return result, nil
}
