package session

import (
	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/configuration"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/sync"
)

// ConfigurationSourceType specifies the source and type of a Configuration
// object. Knowledge of this information is required to appropriately validate a
// Configuration object.
type ConfigurationSourceType uint8

const (
	// ConfigurationSourceTypeSession indicates a session Configuration object
	// sourced from a Session object stored on disk.
	ConfigurationSourceTypeSession ConfigurationSourceType = iota
	// ConfigurationSourceTypeGlobal indicates a session Configuration object
	// sourced from the global configuration file.
	ConfigurationSourceTypeGlobal
	// ConfigurationSourceTypeCreate indicates a session Configuration object
	// sourced from a create RPC request.
	ConfigurationSourceTypeCreate
	// ConfigurationSourceTypeSessionEndpointSpecific indicates an endpoint-
	// specific session Configuration object sourced from a Session object
	// stored on disk.
	ConfigurationSourceTypeSessionEndpointSpecific
	// ConfigurationSourceTypeCreateEndpointSpecific indicates an endpoint-
	// specific session Configuration object sourced from a create RPC request.
	ConfigurationSourceTypeCreateEndpointSpecific
	// ConfigurationSourceTypeAPIEndpointSpecific indicates an endpoint-specific
	// session Configuration object provided directly to an endpoint.
	ConfigurationSourceTypeAPIEndpointSpecific
)

// EnsureValid ensures that Configuration's invariants are respected.
func (c *Configuration) EnsureValid(source ConfigurationSourceType) error {
	// A nil configuration is not considered valid.
	if c == nil {
		return errors.New("nil configuration")
	}

	// Determine whether or not this is an endpoint-specific Configuration
	// object.
	endpointSpecific := source == ConfigurationSourceTypeSessionEndpointSpecific ||
		source == ConfigurationSourceTypeCreateEndpointSpecific ||
		source == ConfigurationSourceTypeAPIEndpointSpecific

	// Validate the synchronization mode.
	if endpointSpecific {
		if !c.SynchronizationMode.IsDefault() {
			return errors.New("synchronization mode cannot be specified on an endpoint-specific basis")
		}
	} else {
		if !(c.SynchronizationMode.IsDefault() || c.SynchronizationMode.Supported()) {
			return errors.New("unknown or unsupported synchronization mode")
		}
	}

	// The maximum entry count doesn't need to be validated - any of its values
	// are technically valid regardless of the source.

	// The maximum staging file size doesn't need to be validated - any of its
	// values are technically valid regardless of the source.

	// Verify that the symlink mode.
	if endpointSpecific {
		if !c.SymlinkMode.IsDefault() {
			return errors.New("symbolic link handling mode cannot be specified on an endpoint-specific basis")
		}
	} else {
		if !(c.SymlinkMode.IsDefault() || c.SymlinkMode.Supported()) {
			return errors.New("unknown or unsupported symlink mode")
		}
	}

	// Verify that the watch mode is unspecified or supported for usage.
	if !(c.WatchMode.IsDefault() || c.WatchMode.Supported()) {
		return errors.New("unknown or unsupported watch mode")
	}

	// The watch polling interval doesn't need to be validated - any of its
	// values are technically valid regardless of the source.

	// Verify that default ignores are unset, unless this is a Configuration
	// object sourced from an existing Session. If there are any allowed
	// DefaultIgnores, verify that they're valid. This field is deprecated and
	// no longer used.
	if source != ConfigurationSourceTypeSession && len(c.DefaultIgnores) > 0 {
		return errors.New("deprecated default ignores configuration field specified")
	}
	for _, ignore := range c.DefaultIgnores {
		if !sync.ValidIgnorePattern(ignore) {
			return errors.Errorf("invalid default ignore pattern: %s", ignore)
		}
	}

	// Verify that ignores are unset for endpoint-specific configurations and
	// that any specified ignores are valid.
	if endpointSpecific && len(c.Ignores) > 0 {
		return errors.New("ignores cannot be specified on an endpoint-specific basis")
	}
	for _, ignore := range c.Ignores {
		if !sync.ValidIgnorePattern(ignore) {
			return errors.Errorf("invalid ignore pattern: %s", ignore)
		}
	}

	// Verify that the VCS ignore mode is unspecified or supported for usage.
	if endpointSpecific {
		if !c.IgnoreVCSMode.IsDefault() {
			return errors.New("VCS ignore mode cannot be specified on an endpoint-specific basis")
		}
	} else {
		if !(c.IgnoreVCSMode.IsDefault() || c.IgnoreVCSMode.Supported()) {
			return errors.New("unknown or unsupported VCS ignore mode")
		}
	}

	// Verify the default file mode.
	if c.DefaultFileMode != 0 {
		if err := sync.EnsureDefaultFileModeValid(filesystem.Mode(c.DefaultFileMode)); err != nil {
			return errors.Wrap(err, "invalid default file permission mode specified")
		}
	}

	// Verify the default directory mode.
	if c.DefaultDirectoryMode != 0 {
		if err := sync.EnsureDefaultDirectoryModeValid(filesystem.Mode(c.DefaultDirectoryMode)); err != nil {
			return errors.Wrap(err, "invalid default directory permission mode specified")
		}
	}

	// Verify the default owner specification.
	if c.DefaultOwner != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(c.DefaultOwner); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid default owner specification")
		}
	}

	// Verify the default group specification.
	if c.DefaultGroup != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(c.DefaultGroup); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid default group specification")
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

	// Create a session configuration object.
	result := &Configuration{
		SynchronizationMode:    configuration.Synchronization.Mode,
		MaximumEntryCount:      configuration.Synchronization.MaximumEntryCount,
		MaximumStagingFileSize: uint64(configuration.Synchronization.MaximumStagingFileSize),
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
	if err := result.EnsureValid(ConfigurationSourceTypeGlobal); err != nil {
		return nil, errors.Wrap(err, "global configuration invalid")
	}

	// Success.
	return result, nil
}

// MergeConfigurations merges two configurations of differing priorities. Both
// configurations must be non-nil.
func MergeConfigurations(lower, higher *Configuration) *Configuration {
	// Create the resulting configuration.
	result := &Configuration{}

	// Merge synchronization mode.
	if !higher.SynchronizationMode.IsDefault() {
		result.SynchronizationMode = higher.SynchronizationMode
	} else {
		result.SynchronizationMode = lower.SynchronizationMode
	}

	// Merge maximum entry count.
	if higher.MaximumEntryCount != 0 {
		result.MaximumEntryCount = higher.MaximumEntryCount
	} else {
		result.MaximumEntryCount = lower.MaximumEntryCount
	}

	// Merge maximum staging file size.
	if higher.MaximumStagingFileSize != 0 {
		result.MaximumStagingFileSize = higher.MaximumStagingFileSize
	} else {
		result.MaximumStagingFileSize = lower.MaximumStagingFileSize
	}

	// Merge symlink mode.
	if !higher.SymlinkMode.IsDefault() {
		result.SymlinkMode = higher.SymlinkMode
	} else {
		result.SymlinkMode = lower.SymlinkMode
	}

	// Merge watch mode.
	if !higher.WatchMode.IsDefault() {
		result.WatchMode = higher.WatchMode
	} else {
		result.WatchMode = lower.WatchMode
	}

	// Merge polling interval.
	if higher.WatchPollingInterval != 0 {
		result.WatchPollingInterval = higher.WatchPollingInterval
	} else {
		result.WatchPollingInterval = lower.WatchPollingInterval
	}

	// Merge default ignores. In theory, at most one of these should be
	// non-empty, but we'll still implement it as if they both might have
	// content.
	result.DefaultIgnores = append(result.DefaultIgnores, lower.DefaultIgnores...)
	result.DefaultIgnores = append(result.DefaultIgnores, higher.DefaultIgnores...)

	// Merge ignores.
	result.Ignores = append(result.Ignores, lower.Ignores...)
	result.Ignores = append(result.Ignores, higher.Ignores...)

	// Merge VCS ignore mode.
	if !higher.IgnoreVCSMode.IsDefault() {
		result.IgnoreVCSMode = higher.IgnoreVCSMode
	} else {
		result.IgnoreVCSMode = lower.IgnoreVCSMode
	}

	// Merge default file mode.
	if higher.DefaultFileMode != 0 {
		result.DefaultFileMode = higher.DefaultFileMode
	} else {
		result.DefaultFileMode = lower.DefaultFileMode
	}

	// Merge default directory mode.
	if higher.DefaultDirectoryMode != 0 {
		result.DefaultDirectoryMode = higher.DefaultDirectoryMode
	} else {
		result.DefaultDirectoryMode = lower.DefaultDirectoryMode
	}

	// Merge default owner.
	if higher.DefaultOwner != "" {
		result.DefaultOwner = higher.DefaultOwner
	} else {
		result.DefaultOwner = lower.DefaultOwner
	}

	// Merge default group.
	if higher.DefaultGroup != "" {
		result.DefaultGroup = higher.DefaultGroup
	} else {
		result.DefaultGroup = lower.DefaultGroup
	}

	// Done.
	return result
}
