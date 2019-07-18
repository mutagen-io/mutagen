package legacy

import (
	"github.com/mutagen-io/mutagen/pkg/configuration/types"
	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// Configuration is the legacy TOML-based Mutagen session configuration format.
type Configuration struct {
	// Synchronization contains parameters related to synchronization behavior.
	Synchronization struct {
		// Mode specifies the default synchronization mode.
		Mode core.SynchronizationMode `toml:"mode"`
		// MaximumEntryCount specifies the maximum number of filesystem entries
		// that endpoints will tolerate managing.
		MaximumEntryCount uint64 `toml:"maxEntryCount"`
		// MaximumStagingFileSize is the maximum (individual) file size that
		// endpoints will stage. It can be specified in human-friendly units.
		MaximumStagingFileSize types.ByteSize `toml:"maxStagingFileSize"`
		// ProbeMode specifies the filesystem probing mode.
		ProbeMode behavior.ProbeMode `toml:"probeMode"`
		// ScanMode specifies the filesystem scanning mode.
		ScanMode synchronization.ScanMode `toml:"scanMode"`
		// StageMode specifies the filesystem staging mode.
		StageMode synchronization.StageMode `toml:"stageMode"`
	} `toml:"sync"`
	// Ignore contains parameters related to synchronization ignore
	// specifications.
	Ignore struct {
		// Default specifies the default list of ignore specifications.
		Default []string `toml:"default"`
		// VCS specifies the VCS ignore mode.
		VCS core.IgnoreVCSMode `toml:"vcs"`
	} `toml:"ignore"`
	// Symlink contains parameters related to symlink handling.
	Symlink struct {
		// Mode specifies the symlink mode.
		Mode core.SymlinkMode `toml:"mode"`
	} `toml:"symlink"`
	// Watch contains parameters related to filesystem monitoring.
	Watch struct {
		// Mode specifies the file watching mode.
		Mode synchronization.WatchMode `toml:"mode"`
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

// Configuration converts a legacy TOML-based session configuration to a
// Protocol Buffers session configuration. It does not validate the resulting
// configuration.
func (c *Configuration) Configuration() *synchronization.Configuration {
	return &synchronization.Configuration{
		SynchronizationMode:    c.Synchronization.Mode,
		MaximumEntryCount:      c.Synchronization.MaximumEntryCount,
		MaximumStagingFileSize: uint64(c.Synchronization.MaximumStagingFileSize),
		ProbeMode:              c.Synchronization.ProbeMode,
		ScanMode:               c.Synchronization.ScanMode,
		StageMode:              c.Synchronization.StageMode,
		SymlinkMode:            c.Symlink.Mode,
		WatchMode:              c.Watch.Mode,
		WatchPollingInterval:   c.Watch.PollingInterval,
		Ignores:                c.Ignore.Default,
		IgnoreVCSMode:          c.Ignore.VCS,
		DefaultFileMode:        uint32(c.Permissions.DefaultFileMode),
		DefaultDirectoryMode:   uint32(c.Permissions.DefaultDirectoryMode),
		DefaultOwner:           c.Permissions.DefaultOwner,
		DefaultGroup:           c.Permissions.DefaultGroup,
	}
}

// LoadConfiguration attempts to load a legacy TOML-based Mutagen
// synchronization configuration file from the specified path.
func LoadConfiguration(path string) (*Configuration, error) {
	// Create the target configuration object.
	result := &Configuration{}

	// Attempt to load. We pass-through os.IsNotExist errors.
	if err := encoding.LoadAndUnmarshalTOML(path, result); err != nil {
		return nil, err
	}

	// Success.
	return result, nil
}
