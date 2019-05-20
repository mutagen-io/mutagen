package configuration

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/session"
)

// LoadSessionConfiguration loads a TOML-based Mutagen configuration and
// extracts a corresponding Mutagen session configuration, verifying that it's a
// valid non-endpoint-specific session configuration.
func LoadSessionConfiguration(path string) (*session.Configuration, error) {
	// Load the TOML-based configuration.
	configuration, err := Load(path)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load TOML-based configuration")
	}

	// Create a session configuration object.
	result := &session.Configuration{
		SynchronizationMode:    configuration.Synchronization.Mode,
		MaximumEntryCount:      configuration.Synchronization.MaximumEntryCount,
		MaximumStagingFileSize: uint64(configuration.Synchronization.MaximumStagingFileSize),
		ProbeMode:              configuration.Synchronization.ProbeMode,
		ScanMode:               configuration.Synchronization.ScanMode,
		StageMode:              configuration.Synchronization.StageMode,
		SymlinkMode:            configuration.Symlink.Mode,
		WatchMode:              configuration.Watch.Mode,
		WatchPollingInterval:   configuration.Watch.PollingInterval,
		Ignores:                configuration.Ignore.Default,
		IgnoreVCSMode:          configuration.Ignore.VCS,
		DefaultFileMode:        uint32(configuration.Permissions.DefaultFileMode),
		DefaultDirectoryMode:   uint32(configuration.Permissions.DefaultDirectoryMode),
		DefaultOwner:           configuration.Permissions.DefaultOwner,
		DefaultGroup:           configuration.Permissions.DefaultGroup,
	}

	// Verify that the resulting configuration is valid.
	if err := result.EnsureValid(false); err != nil {
		return nil, errors.Wrap(err, "configuration invalid")
	}

	// Success.
	return result, nil
}
