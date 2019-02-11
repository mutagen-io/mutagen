package session

import (
	"github.com/pkg/errors"

	"github.com/dustin/go-humanize"

	"github.com/havoc-io/mutagen/pkg/configuration"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// ConfigurationSource represents the source of a configuration object.
type ConfigurationSource uint8

const (
	// ConfigurationSourceSession specifies that a configuration object came
	// from a session object stored on disk.
	ConfigurationSourceSession ConfigurationSource = iota
	// ConfigurationSourceGlobal specifies that a configuration object was
	// loaded from the global configuration file.
	ConfigurationSourceGlobal
	// ConfigurationSourceCreate specifies that a configuration object came from
	// a create RPC request.
	ConfigurationSourceCreate
)

// EnsureValid ensures that Configuration's invariants are respected.
func (c *Configuration) EnsureValid(source ConfigurationSource) error {
	// A nil configuration is not considered valid.
	if c == nil {
		return errors.New("nil configuration")
	}

	// Verify that the synchronization mode is unspecified or supported for
	// usage.
	if !c.SynchronizationMode.IsDefault() && !c.SynchronizationMode.Supported() {
		return errors.New("unknown or unsupported synchronization mode")
	}

	// The maximum entry count doesn't need to be validated - any of its values
	// are technically valid.

	// The maximum staging file size doesn't need to be validated - any of its
	// values are technically valid.

	// Verify that the symlink mode is unspecified or supported for usage.
	if c.SymlinkMode != sync.SymlinkMode_SymlinkDefault && !c.SymlinkMode.Supported() {
		return errors.New("unknown or unsupported symlink mode")
	}

	// Verify that the watch mode is unspecified or supported for usage.
	if !c.WatchMode.IsDefault() && !c.WatchMode.Supported() {
		return errors.New("unknown or unsupported watch mode")
	}

	// The watch polling interval doesn't need to be validated - any of its
	// values are technically valid.

	// Verify that default ignores are allowed to be specified and that all
	// specified default ignores are valid.
	if source == ConfigurationSourceCreate && len(c.DefaultIgnores) > 0 {
		return errors.New("create configuration with default ignores specified")
	}
	for _, ignore := range c.DefaultIgnores {
		if !sync.ValidIgnorePattern(ignore) {
			return errors.Errorf("invalid default ignore pattern: %s", ignore)
		}
	}

	// Verify that ignores are allowed to be specified and that all specified
	// ignores are valid.
	if source == ConfigurationSourceGlobal && len(c.Ignores) > 0 {
		return errors.New("global configuration with ignores specified")
	}
	for _, ignore := range c.Ignores {
		if !sync.ValidIgnorePattern(ignore) {
			return errors.Errorf("invalid ignore pattern: %s", ignore)
		}
	}

	// Verify that the VCS ignore mode is unspecified or supported for usage.
	if c.IgnoreVCSMode != sync.IgnoreVCSMode_IgnoreVCSDefault && !c.IgnoreVCSMode.Supported() {
		return errors.New("unknown or unsupported VCS ignore mode")
	}

	// Verify that permission settings are empty from the global configuration
	// and sane from the per-session configuration.
	if source == ConfigurationSourceGlobal {
		// Verify that default file permission modes are all 0.
		if c.PermissionDefaultFileMode != 0 {
			return errors.New("global configuration with default file permission mode specified")
		} else if c.PermissionDefaultFileModeAlpha != 0 {
			return errors.New("global configuration with alpha-specific default file permission mode specified")
		} else if c.PermissionDefaultFileModeBeta != 0 {
			return errors.New("global configuration with beta-specific default file permission mode specified")
		}

		// Verify that default directory permission modes are all 0.
		if c.PermissionDefaultDirectoryMode != 0 {
			return errors.New("global configuration with default directory permission mode specified")
		} else if c.PermissionDefaultDirectoryModeAlpha != 0 {
			return errors.New("global configuration with alpha-specific default directory permission mode specified")
		} else if c.PermissionDefaultDirectoryModeBeta != 0 {
			return errors.New("global configuration with beta-specific default directory permission mode specified")
		}

		// Verify that default owner user identifiers are all empty.
		if c.PermissionDefaultUser != "" {
			return errors.New("global configuration with default owner user identifier specified")
		} else if c.PermissionDefaultUserAlpha != "" {
			return errors.New("global configuration with alpha-specific default owner user identifier specified")
		} else if c.PermissionDefaultUserBeta != "" {
			return errors.New("global configuration with beta-specific default owner user identifier specified")
		}

		// Verify that default owner group identifiers are all empty.
		if c.PermissionDefaultGroup != "" {
			return errors.New("global configuration with default owner group identifier specified")
		} else if c.PermissionDefaultGroupAlpha != "" {
			return errors.New("global configuration with alpha-specific default owner group identifier specified")
		} else if c.PermissionDefaultGroupBeta != "" {
			return errors.New("global configuration with beta-specific default owner group identifier specified")
		}
	} else {
		// Verify that default file permission modes are all 0 or valid.
		if c.PermissionDefaultFileMode != 0 {
			if err := sync.EnsureDefaultFileModeValid(filesystem.Mode(c.PermissionDefaultFileMode)); err != nil {
				return errors.Wrap(err, "invalid default file permission mode specified")
			}
		}
		if c.PermissionDefaultFileModeAlpha != 0 {
			if err := sync.EnsureDefaultFileModeValid(filesystem.Mode(c.PermissionDefaultFileModeAlpha)); err != nil {
				return errors.Wrap(err, "invalid alpha-specific default file permission mode specified")
			}
		}
		if c.PermissionDefaultFileModeBeta != 0 {
			if err := sync.EnsureDefaultFileModeValid(filesystem.Mode(c.PermissionDefaultFileModeBeta)); err != nil {
				return errors.Wrap(err, "invalid beta-specific default file permission mode specified")
			}
		}

		// Verify that default directory permission modes are all 0 or valid.
		if c.PermissionDefaultDirectoryMode != 0 {
			if err := sync.EnsureDefaultDirectoryModeValid(filesystem.Mode(c.PermissionDefaultDirectoryMode)); err != nil {
				return errors.Wrap(err, "invalid default directory permission mode specified")
			}
		}
		if c.PermissionDefaultDirectoryModeAlpha != 0 {
			if err := sync.EnsureDefaultDirectoryModeValid(filesystem.Mode(c.PermissionDefaultDirectoryModeAlpha)); err != nil {
				return errors.Wrap(err, "invalid alpha-specific default directory permission mode specified")
			}
		}
		if c.PermissionDefaultDirectoryModeBeta != 0 {
			if err := sync.EnsureDefaultDirectoryModeValid(filesystem.Mode(c.PermissionDefaultDirectoryModeBeta)); err != nil {
				return errors.Wrap(err, "invalid beta-specific default directory permission mode specified")
			}
		}

		// Verify that default owner user identifiers are all empty or valid.
		if c.PermissionDefaultUser != "" {
			if kind, _ := filesystem.ParseOwnershipIdentifier(c.PermissionDefaultUser); kind == filesystem.OwnershipIdentifierKindInvalid {
				return errors.New("invalid default owner user identifier specified")
			}
		}
		if c.PermissionDefaultUserAlpha != "" {
			if kind, _ := filesystem.ParseOwnershipIdentifier(c.PermissionDefaultUserAlpha); kind == filesystem.OwnershipIdentifierKindInvalid {
				return errors.New("invalid alpha-specific default owner user identifier specified")
			}
		}
		if c.PermissionDefaultUserBeta != "" {
			if kind, _ := filesystem.ParseOwnershipIdentifier(c.PermissionDefaultUserBeta); kind == filesystem.OwnershipIdentifierKindInvalid {
				return errors.New("invalid beta-specific default owner user identifier specified")
			}
		}

		// Verify that default owner group identifiers are all empty or valid.
		if c.PermissionDefaultGroup != "" {
			if kind, _ := filesystem.ParseOwnershipIdentifier(c.PermissionDefaultGroup); kind == filesystem.OwnershipIdentifierKindInvalid {
				return errors.New("invalid default owner group identifier specified")
			}
		}
		if c.PermissionDefaultGroupAlpha != "" {
			if kind, _ := filesystem.ParseOwnershipIdentifier(c.PermissionDefaultGroupAlpha); kind == filesystem.OwnershipIdentifierKindInvalid {
				return errors.New("invalid alpha-specific default owner group identifier specified")
			}
		}
		if c.PermissionDefaultGroupBeta != "" {
			if kind, _ := filesystem.ParseOwnershipIdentifier(c.PermissionDefaultGroupBeta); kind == filesystem.OwnershipIdentifierKindInvalid {
				return errors.New("invalid beta-specific default owner group identifier specified")
			}
		}
	}

	// Success.
	return nil
}

// snapshotGlobalConfiguration loads the global configuration, transfers the
// relevant parameters to a session configuration, and returns the resulting
// value. It does not validate the
func snapshotGlobalConfiguration() (*Configuration, error) {
	// Load the global configuration.
	configuration, err := configuration.Load()
	if err != nil {
		return nil, errors.Wrap(err, "unable to load global configuration")
	}

	// Parse maximum staging file size if specified.
	var maximumStagingFileSize uint64
	if configuration.Synchronization.MaximumStagingFileSize != "" {
		if s, err := humanize.ParseBytes(configuration.Synchronization.MaximumStagingFileSize); err != nil {
			return nil, errors.Wrap(err, "unable to parse maximum staging file size")
		} else {
			maximumStagingFileSize = s
		}
	}

	// Create a session configuration object.
	result := &Configuration{
		SynchronizationMode:    configuration.Synchronization.Mode,
		MaximumEntryCount:      configuration.Synchronization.MaximumEntryCount,
		MaximumStagingFileSize: maximumStagingFileSize,
		SymlinkMode:            configuration.Symlink.Mode,
		WatchMode:              configuration.Watch.Mode,
		WatchPollingInterval:   configuration.Watch.PollingInterval,
		DefaultIgnores:         configuration.Ignore.Default,
		IgnoreVCSMode:          configuration.Ignore.VCS,
	}

	// Verify that the resulting configuration is valid.
	if err := result.EnsureValid(ConfigurationSourceGlobal); err != nil {
		return nil, errors.Wrap(err, "global configuration invalid")
	}

	// Success.
	return result, nil
}

// MergeConfigurations merges a per-session and global configuration, allowing
// the per-session configuration to merge with or override the global
// configuration.
func MergeConfigurations(session, global *Configuration) *Configuration {
	// Create the resulting configuration.
	result := &Configuration{}

	// Merge synchronization mode.
	if !session.SynchronizationMode.IsDefault() {
		result.SynchronizationMode = session.SynchronizationMode
	} else {
		result.SynchronizationMode = global.SynchronizationMode
	}

	// Merge maximum entry count.
	if session.MaximumEntryCount != 0 {
		result.MaximumEntryCount = session.MaximumEntryCount
	} else {
		result.MaximumEntryCount = global.MaximumEntryCount
	}

	// Merge maximum staging file size.
	if session.MaximumStagingFileSize != 0 {
		result.MaximumStagingFileSize = session.MaximumStagingFileSize
	} else {
		result.MaximumStagingFileSize = global.MaximumStagingFileSize
	}

	// Merge symlink mode.
	if session.SymlinkMode != sync.SymlinkMode_SymlinkDefault {
		result.SymlinkMode = session.SymlinkMode
	} else {
		result.SymlinkMode = global.SymlinkMode
	}

	// Merge watch mode.
	if session.WatchMode != filesystem.WatchMode_WatchModeDefault {
		result.WatchMode = session.WatchMode
	} else {
		result.WatchMode = global.WatchMode
	}

	// Merge polling interval.
	if session.WatchPollingInterval != 0 {
		result.WatchPollingInterval = session.WatchPollingInterval
	} else {
		result.WatchPollingInterval = global.WatchPollingInterval
	}

	// Propagate default ignores.
	result.DefaultIgnores = global.DefaultIgnores

	// Propagate per-session ignores.
	result.Ignores = session.Ignores

	// Merge VCS ignore mode.
	if session.IgnoreVCSMode != sync.IgnoreVCSMode_IgnoreVCSDefault {
		result.IgnoreVCSMode = session.IgnoreVCSMode
	} else {
		result.IgnoreVCSMode = global.IgnoreVCSMode
	}

	// Merge default file permission modes. These are all currently disallowed
	// in global configuration.
	result.PermissionDefaultFileMode = session.PermissionDefaultFileMode
	result.PermissionDefaultFileModeAlpha = session.PermissionDefaultFileModeAlpha
	result.PermissionDefaultFileModeBeta = session.PermissionDefaultFileModeBeta

	// Merge default directory permission modes. These are all currently
	// disallowed in global configuration.
	result.PermissionDefaultDirectoryMode = session.PermissionDefaultDirectoryMode
	result.PermissionDefaultDirectoryModeAlpha = session.PermissionDefaultDirectoryModeAlpha
	result.PermissionDefaultDirectoryModeBeta = session.PermissionDefaultDirectoryModeBeta

	// Merge default owner user identifiers. These are all currently disallowed
	// in global configuration.
	result.PermissionDefaultUser = session.PermissionDefaultUser
	result.PermissionDefaultUserAlpha = session.PermissionDefaultUserAlpha
	result.PermissionDefaultUserBeta = session.PermissionDefaultUserBeta

	// Merge default owner group identifiers. These are all currently disallowed
	// in global configuration.
	result.PermissionDefaultGroup = session.PermissionDefaultGroup
	result.PermissionDefaultGroupAlpha = session.PermissionDefaultGroupAlpha
	result.PermissionDefaultGroupBeta = session.PermissionDefaultGroupBeta

	// Done.
	return result
}
