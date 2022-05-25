package synchronization

import (
	"github.com/mutagen-io/mutagen/pkg/api/models/types"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// Configuration represents synchronization session configuration.
type Configuration struct {
	// Mode specifies the default synchronization mode.
	Mode core.SynchronizationMode `json:"mode,omitempty" yaml:"mode" mapstructure:"mode"`
	// MaximumEntryCount specifies the maximum number of filesystem entries
	// that endpoints will tolerate managing.
	MaximumEntryCount uint64 `json:"maxEntryCount,omitempty" yaml:"maxEntryCount" mapstructure:"maxEntryCount"`
	// MaximumStagingFileSize is the maximum (individual) file size that
	// endpoints will stage. It can be specified in human-friendly units.
	MaximumStagingFileSize types.ByteSize `json:"maxStagingFileSize,omitempty" yaml:"maxStagingFileSize" mapstructure:"maxStagingFileSize"`
	// ProbeMode specifies the filesystem probing mode.
	ProbeMode behavior.ProbeMode `json:"probeMode,omitempty" yaml:"probeMode" mapstructure:"probeMode"`
	// ScanMode specifies the filesystem scanning mode.
	ScanMode synchronization.ScanMode `json:"scanMode,omitempty" yaml:"scanMode" mapstructure:"scanMode"`
	// StageMode specifies the filesystem staging mode.
	StageMode synchronization.StageMode `json:"stageMode,omitempty" yaml:"stageMode" mapstructure:"stageMode"`
	// Ignore contains parameters related to synchronization ignore
	// specifications.
	Ignore struct {
		// Paths specifies the default list of ignore specifications.
		Paths []string `json:"paths,omitempty" yaml:"paths" mapstructure:"paths"`
		// VCS specifies the VCS ignore mode.
		VCS core.IgnoreVCSMode `json:"vcs,omitempty" yaml:"vcs" mapstructure:"vcs"`
	} `json:"ignore" yaml:"ignore" mapstructure:"ignore"`
	// Symlink contains parameters related to symbolic link handling.
	Symlink struct {
		// Mode specifies the symbolic link mode.
		Mode core.SymbolicLinkMode `json:"mode,omitempty" yaml:"mode" mapstructure:"mode"`
	} `json:"symlink" yaml:"symlink" mapstructure:"symlink"`
	// Watch contains parameters related to filesystem monitoring.
	Watch struct {
		// Mode specifies the file watching mode.
		Mode synchronization.WatchMode `json:"mode,omitempty" yaml:"mode" mapstructure:"mode"`
		// PollingInterval specifies the interval (in seconds) for poll-based
		// file monitoring. A value of 0 specifies that Mutagen's internal
		// default interval should be used.
		PollingInterval uint32 `json:"pollingInterval,omitempty" yaml:"pollingInterval" mapstructure:"pollingInterval"`
	} `json:"watch" yaml:"watch" mapstructure:"watch"`
	// Permissions contains parameters related to permission handling.
	Permissions struct {
		// DefaultFileMode specifies the default permission mode to use for new
		// files in "portable" permission propagation mode.
		DefaultFileMode filesystem.Mode `json:"defaultFileMode,omitempty" yaml:"defaultFileMode" mapstructure:"defaultFileMode"`
		// DefaultDirectoryMode specifies the default permission mode to use for
		// new files in "portable" permission propagation mode.
		DefaultDirectoryMode filesystem.Mode `json:"defaultDirectoryMode,omitempty" yaml:"defaultDirectoryMode" mapstructure:"defaultDirectoryMode"`
		// DefaultOwner specifies the default owner identifier to use when
		// setting ownership of new files and directories in "portable"
		// permission propagation mode.
		DefaultOwner string `json:"defaultOwner,omitempty" yaml:"defaultOwner" mapstructure:"defaultOwner"`
		// DefaultGroup specifies the default group identifier to use when
		// setting ownership of new files and directories in "portable"
		// permission propagation mode.
		DefaultGroup string `json:"defaultGroup,omitempty" yaml:"defaultGroup" mapstructure:"defaultGroup"`
	} `json:"permissions" yaml:"permissions" mapstructure:"permissions"`
}

// NewConfigurationFromInternalConfiguration creates a new configuration
// representation from an internal Protocol Buffers representation. The
// configuration must be valid.
func NewConfigurationFromInternalConfiguration(configuration *synchronization.Configuration) *Configuration {
	// Create the result.
	result := &Configuration{}

	// Propagate top-level configuration.
	result.Mode = configuration.SynchronizationMode
	result.MaximumEntryCount = configuration.MaximumEntryCount
	result.MaximumStagingFileSize = types.ByteSize(configuration.MaximumStagingFileSize)
	result.ProbeMode = configuration.ProbeMode
	result.ScanMode = configuration.ScanMode
	result.StageMode = configuration.StageMode

	// Propagate ignore configuration.
	result.Ignore.Paths = append(result.Ignore.Paths, configuration.DefaultIgnores...)
	result.Ignore.Paths = append(result.Ignore.Paths, configuration.Ignores...)
	result.Ignore.VCS = configuration.IgnoreVCSMode

	// Propagate symbolic link configuration.
	result.Symlink.Mode = configuration.SymbolicLinkMode

	// Propagate watch configuration.
	result.Watch.Mode = configuration.WatchMode
	result.Watch.PollingInterval = configuration.WatchPollingInterval

	// Propagate permission configuration.
	result.Permissions.DefaultFileMode = filesystem.Mode(configuration.DefaultFileMode)
	result.Permissions.DefaultDirectoryMode = filesystem.Mode(configuration.DefaultDirectoryMode)
	result.Permissions.DefaultOwner = configuration.DefaultOwner
	result.Permissions.DefaultGroup = configuration.DefaultGroup

	// Done.
	return result
}

// ToInternalConfiguration converts a textual session configuration to an
// internal Protocol Buffers session configuration. It does not validate the
// resulting configuration.
func (c *Configuration) ToInternalConfiguration() *synchronization.Configuration {
	return &synchronization.Configuration{
		SynchronizationMode:    c.Mode,
		MaximumEntryCount:      c.MaximumEntryCount,
		MaximumStagingFileSize: uint64(c.MaximumStagingFileSize),
		ProbeMode:              c.ProbeMode,
		ScanMode:               c.ScanMode,
		StageMode:              c.StageMode,
		SymbolicLinkMode:       c.Symlink.Mode,
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
