package synchronization

import (
	"github.com/pkg/errors"

	"github.com/dustin/go-humanize"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// ByteSize is a uint64 value that supports unmarshalling from both
// human-friendly string representations and numeric representations. It can be
// cast to a uint64 value, where it represents a byte count.
type ByteSize uint64

// UnmarshalText implements the text unmarshalling interface used when loading
// from TOML files.
func (s *ByteSize) UnmarshalText(textBytes []byte) error {
	// Convert the bytes to a string.
	text := string(textBytes)

	// Parse and store the value.
	value, err := humanize.ParseBytes(text)
	if err != nil {
		return err
	}
	*s = ByteSize(value)

	// Success.
	return nil
}

// YAMLConfiguration represents a human-readable Mutagen session configuration,
// loadable from either TOML or YAML.
type YAMLConfiguration struct {
	// Mode specifies the default synchronization mode.
	Mode core.SynchronizationMode `yaml:"mode"`
	// MaximumEntryCount specifies the maximum number of filesystem entries
	// that endpoints will tolerate managing.
	MaximumEntryCount uint64 `yaml:"maxEntryCount"`
	// MaximumStagingFileSize is the maximum (individual) file size that
	// endpoints will stage. It can be specified in human-friendly units.
	MaximumStagingFileSize ByteSize `yaml:"maxStagingFileSize"`
	// ProbeMode specifies the filesystem probing mode.
	ProbeMode behavior.ProbeMode `yaml:"probeMode"`
	// ScanMode specifies the filesystem scanning mode.
	ScanMode ScanMode `yaml:"scanMode"`
	// StageMode specifies the filesystem staging mode.
	StageMode StageMode `yaml:"stageMode"`
	// Ignore contains parameters related to synchronization ignore
	// specifications.
	Ignore struct {
		// Paths specifies the default list of ignore specifications.
		Paths []string `yaml:"paths"`
		// VCS specifies the VCS ignore mode.
		VCS core.IgnoreVCSMode `yaml:"vcs"`
	} `yaml:"ignore"`
	// Symlink contains parameters related to symlink handling.
	Symlink struct {
		// Mode specifies the symlink mode.
		Mode core.SymlinkMode `yaml:"mode"`
	} `yaml:"symlink"`
	// Watch contains parameters related to filesystem monitoring.
	Watch struct {
		// Mode specifies the file watching mode.
		Mode WatchMode `yaml:"mode"`
		// PollingInterval specifies the interval (in seconds) for poll-based
		// file monitoring. A value of 0 specifies that Mutagen's internal
		// default interval should be used.
		PollingInterval uint32 `yaml:"pollingInterval"`
	} `yaml:"watch"`
	// Permissions contains parameters related to permission handling.
	Permissions struct {
		// DefaultFileMode specifies the default permission mode to use for new
		// files in "portable" permission propagation mode.
		DefaultFileMode filesystem.Mode `yaml:"defaultFileMode"`
		// DefaultDirectoryMode specifies the default permission mode to use for
		// new files in "portable" permission propagation mode.
		DefaultDirectoryMode filesystem.Mode `yaml:"defaultDirectoryMode"`
		// DefaultOwner specifies the default owner identifier to use when
		// setting ownership of new files and directories in "portable"
		// permission propagation mode.
		DefaultOwner string `yaml:"defaultOwner"`
		// DefaultGroup specifies the default group identifier to use when
		// setting ownership of new files and directories in "portable"
		// permission propagation mode.
		DefaultGroup string `yaml:"defaultGroup"`
	} `yaml:"permissions"`
}

// Configuration converts a human-readable session configuration to a Protocol
// Buffers session configuration. It does not validate the resulting
// configuration.
func (c *YAMLConfiguration) Configuration() *Configuration {
	return &Configuration{
		SynchronizationMode:    c.Mode,
		MaximumEntryCount:      c.MaximumEntryCount,
		MaximumStagingFileSize: uint64(c.MaximumStagingFileSize),
		ProbeMode:              c.ProbeMode,
		ScanMode:               c.ScanMode,
		StageMode:              c.StageMode,
		SymlinkMode:            c.Symlink.Mode,
		WatchMode:              c.Watch.Mode,
		WatchPollingInterval:   c.Watch.PollingInterval,
		Ignores:                c.Ignore.Paths,
		IgnoreVCSMode:          c.Ignore.VCS,
		DefaultFileMode:        uint32(c.Permissions.DefaultFileMode),
		DefaultDirectoryMode:   uint32(c.Permissions.DefaultDirectoryMode),
		DefaultOwner:           c.Permissions.DefaultOwner,
		DefaultGroup:           c.Permissions.DefaultGroup,
	}
}

// EnsureValid ensures that Configuration's invariants are respected. The
// validation of the configuration depends on whether or not it is
// endpoint-specific.
func (c *Configuration) EnsureValid(endpointSpecific bool) error {
	// A nil configuration is not considered valid.
	if c == nil {
		return errors.New("nil configuration")
	}

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

	// Verify that the probe mode is unspecified or supported for usage.
	if !(c.ProbeMode.IsDefault() || c.ProbeMode.Supported()) {
		return errors.New("unknown or unsupported probe mode")
	}

	// Verify that the scan mode is unspecified or supported for usage.
	if !(c.ScanMode.IsDefault() || c.ScanMode.Supported()) {
		return errors.New("unknown or unsupported scan mode")
	}

	// Verify that the staging mode is unspecified or supported for usage.
	if !(c.StageMode.IsDefault() || c.StageMode.Supported()) {
		return errors.New("unknown or unsupported staging mode")
	}

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

	// Verify that default ignores are unset for endpoint-specific
	// configurations and that any specified ignores are valid. This field is
	// deprecated, but existing sessions may have it set, in which case we'll
	// just prepend it to the nominal list of ignores when running the session.
	// We don't bother rejecting its presence based on source.
	if endpointSpecific && len(c.DefaultIgnores) > 0 {
		return errors.New("default ignores cannot be specified on an endpoint-specific basis (and are deprecated)")
	}
	for _, ignore := range c.DefaultIgnores {
		if !core.ValidIgnorePattern(ignore) {
			return errors.Errorf("invalid default ignore pattern: %s", ignore)
		}
	}

	// Verify that ignores are unset for endpoint-specific configurations and
	// that any specified ignores are valid.
	if endpointSpecific && len(c.Ignores) > 0 {
		return errors.New("ignores cannot be specified on an endpoint-specific basis")
	}
	for _, ignore := range c.Ignores {
		if !core.ValidIgnorePattern(ignore) {
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
		if err := core.EnsureDefaultFileModeValid(filesystem.Mode(c.DefaultFileMode)); err != nil {
			return errors.Wrap(err, "invalid default file permission mode specified")
		}
	}

	// Verify the default directory mode.
	if c.DefaultDirectoryMode != 0 {
		if err := core.EnsureDefaultDirectoryModeValid(filesystem.Mode(c.DefaultDirectoryMode)); err != nil {
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

	// Merge probe mode.
	if !higher.ProbeMode.IsDefault() {
		result.ProbeMode = higher.ProbeMode
	} else {
		result.ProbeMode = lower.ProbeMode
	}

	// Merge scan mode.
	if !higher.ScanMode.IsDefault() {
		result.ScanMode = higher.ScanMode
	} else {
		result.ScanMode = lower.ScanMode
	}

	// Merge staging mode.
	if !higher.StageMode.IsDefault() {
		result.StageMode = higher.StageMode
	} else {
		result.StageMode = lower.StageMode
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
