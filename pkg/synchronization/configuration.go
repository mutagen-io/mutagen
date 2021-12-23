package synchronization

import (
	"errors"
	"fmt"

	"github.com/mutagen-io/mutagen/pkg/comparison"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

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

	// Verify that the symbolic link mode is unspecified or supported for usage.
	if endpointSpecific {
		if !c.SymbolicLinkMode.IsDefault() {
			return errors.New("symbolic link mode cannot be specified on an endpoint-specific basis")
		}
	} else {
		if !(c.SymbolicLinkMode.IsDefault() || c.SymbolicLinkMode.Supported()) {
			return errors.New("unknown or unsupported symbolic link mode")
		}
	}

	// Validate ignorer mode
	if !(c.IgnorerMode.IsDefault() || c.IgnorerMode.Supported()) {
		return errors.New("unknown or unsupported ignorer mode")
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
			return fmt.Errorf("invalid default ignore pattern: %s", ignore)
		}
	}

	// Verify that ignores are unset for endpoint-specific configurations and
	// that any specified ignores are valid.
	if endpointSpecific && len(c.Ignores) > 0 {
		return errors.New("ignores cannot be specified on an endpoint-specific basis")
	}
	for _, ignore := range c.Ignores {
		if !core.ValidIgnorePattern(ignore) {
			return fmt.Errorf("invalid ignore pattern: %s", ignore)
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
			return fmt.Errorf("invalid default file permission mode specified: %w", err)
		}
	}

	// Verify the default directory mode.
	if c.DefaultDirectoryMode != 0 {
		if err := core.EnsureDefaultDirectoryModeValid(filesystem.Mode(c.DefaultDirectoryMode)); err != nil {
			return fmt.Errorf("invalid default directory permission mode specified: %w", err)
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

// Equal returns whether or not the configuration is equivalent to another. The
// result of this method is only valid if both configurations are valid.
func (c *Configuration) Equal(other *Configuration) bool {
	// Ensure that both are non-nil.
	if c == nil || other == nil {
		return false
	}

	// Perform an equivalence check.
	return c.SynchronizationMode == other.SynchronizationMode &&
		c.MaximumEntryCount == other.MaximumEntryCount &&
		c.MaximumStagingFileSize == other.MaximumStagingFileSize &&
		c.ProbeMode == other.ProbeMode &&
		c.ScanMode == other.ScanMode &&
		c.StageMode == other.StageMode &&
		c.SymbolicLinkMode == other.SymbolicLinkMode &&
		c.WatchMode == other.WatchMode &&
		c.WatchPollingInterval == other.WatchPollingInterval &&
		comparison.StringSlicesEqual(c.DefaultIgnores, other.DefaultIgnores) &&
		comparison.StringSlicesEqual(c.Ignores, other.Ignores) &&
		c.IgnorerMode == other.IgnorerMode &&
		c.IgnoreVCSMode == other.IgnoreVCSMode &&
		c.DefaultFileMode == other.DefaultFileMode &&
		c.DefaultDirectoryMode == other.DefaultDirectoryMode &&
		c.DefaultOwner == other.DefaultOwner &&
		c.DefaultGroup == other.DefaultGroup
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

	// Merge probing mode.
	if !higher.ProbeMode.IsDefault() {
		result.ProbeMode = higher.ProbeMode
	} else {
		result.ProbeMode = lower.ProbeMode
	}

	// Merge scanning mode.
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

	// Merge symbolic link mode.
	if !higher.SymbolicLinkMode.IsDefault() {
		result.SymbolicLinkMode = higher.SymbolicLinkMode
	} else {
		result.SymbolicLinkMode = lower.SymbolicLinkMode
	}

	// Merge ignorer mode.
	if !higher.IgnorerMode.IsDefault() {
		result.IgnorerMode = higher.IgnorerMode
	} else {
		result.IgnorerMode = lower.IgnorerMode
	}

	// Merge watching mode.
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
