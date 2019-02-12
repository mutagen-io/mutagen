package configuration

import (
	"os"

	"github.com/havoc-io/mutagen/pkg/encoding"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// Configuration represents the global Mutagen configuration.
type Configuration struct {
	// Synchronization contains parameters related to synchronization behavior.
	Synchronization struct {
		// Mode specifies the default synchronization mode.
		Mode sync.SynchronizationMode `toml:"mode"`
		// MaximumEntryCount specifies the maximum number of filesystem entries
		// that endpoints will tolerate managing.
		MaximumEntryCount uint64 `toml:"maxEntryCount"`
		// MaximumStagingFileSize is the maximum (individual) file size that
		// endpoints will stage. It can be specified in human-friendly units.
		MaximumStagingFileSize ByteSize `toml:"maxStagingFileSize"`
	} `toml:"sync"`

	// Ignore contains parameters related to synchronization ignore
	// specifications.
	Ignore struct {
		// Default specifies the default list of ignore specifications.
		Default []string `toml:"default"`

		// VCS specifies the VCS ignore mode.
		VCS sync.IgnoreVCSMode `toml:"vcs"`
	} `toml:"ignore"`

	// Symlink contains parameters related to symlink handling.
	Symlink struct {
		// Mode specifies the symlink mode.
		Mode sync.SymlinkMode `toml:"mode"`
	} `toml:"symlink"`

	// Watch contains parameters related to filesystem monitoring.
	Watch struct {
		// Mode specifies the file watching mode.
		Mode filesystem.WatchMode `toml:"mode"`

		// PollingInterval specifies the interval (in seconds) for poll-based
		// file monitoring. A value of 0 specifies that Mutagen's internal
		// default interval should be used.
		PollingInterval uint32 `toml:"pollingInterval"`
	} `toml:"watch"`

	// Permissions contains parameters related to permission handling.
	Permissions struct {
		// DefaultFileMode specifies the default permission mode to use for new
		// files in "portable" permission propagation mode.
		DefaultFileMode filesystem.Mode `toml:"defaultFileMode"`

		// DefaultDirectoryMode specifies the default permission mode to use for
		// new files in "portable" permission propagation mode.
		DefaultDirectoryMode filesystem.Mode `toml:"defaultDirectoryMode"`

		// DefaultOwner specifies the default owner identifier to use when
		// setting ownership of new files and directories in "portable"
		// permission propagation mode.
		DefaultOwner string `toml:"defaultOwner"`

		// DefaultGroup specifies the default group identifier to use when
		// setting ownership of new files and directories in "portable"
		// permission propagation mode.
		DefaultGroup string `toml:"defaultGroup"`
	} `toml:"permissions"`
}

// loadFromPath is the internal loading function. We keep it separate from Load
// so that we can get full test coverage using temporary files.
func loadFromPath(path string) (*Configuration, error) {
	// Create a configuration that we can decode into. We set any default values
	// here because nothing will be modified in this structure if the
	// configuration file doesn't exist.
	result := &Configuration{}

	// Attempt to load the configuration from disk. If loading fails due to the
	// path not existing, we return the blank configuration. We don't need to
	// allocate a fresh one in that case since the loader won't have touched it
	// if the file didn't exist.
	// TODO: Should we implement a caching mechanism where we run a stat call
	// and watch for filesystem modification?
	if err := encoding.LoadAndUnmarshalTOML(path, result); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	// Return the configuration.
	return result, nil
}

// Load loads the Mutagen configuration file from disk and populates a
// Configuration structure. If the Mutagen configuration file does not exist,
// this method will return a structure with the default configuration values.
// The returned structure is not re-used, so its members can be freely mutated.
func Load() (*Configuration, error) {
	return loadFromPath(filesystem.MutagenConfigurationPath)
}
