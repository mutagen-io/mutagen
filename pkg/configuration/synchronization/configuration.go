package synchronization

import (
	"github.com/mutagen-io/mutagen/pkg/configuration/types"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// Configuration represents a human-readable Mutagen session configuration,
// loadable from either TOML or YAML.
type Configuration struct {
	// Mode specifies the default synchronization mode.
	Mode core.SynchronizationMode `yaml:"mode" mapstructure:"mode"`
	// MaximumEntryCount specifies the maximum number of filesystem entries
	// that endpoints will tolerate managing.
	MaximumEntryCount uint64 `yaml:"maxEntryCount" mapstructure:"maxEntryCount"`
	// MaximumStagingFileSize is the maximum (individual) file size that
	// endpoints will stage. It can be specified in human-friendly units.
	MaximumStagingFileSize types.ByteSize `yaml:"maxStagingFileSize" mapstructure:"maxStagingFileSize"`
	// ProbeMode specifies the filesystem probing mode.
	ProbeMode behavior.ProbeMode `yaml:"probeMode" mapstructure:"probeMode"`
	// ScanMode specifies the filesystem scanning mode.
	ScanMode synchronization.ScanMode `yaml:"scanMode" mapstructure:"scanMode"`
	// StageMode specifies the filesystem staging mode.
	StageMode synchronization.StageMode `yaml:"stageMode" mapstructure:"stageMode"`
	// Ignore contains parameters related to synchronization ignore
	// specifications.
	Ignore struct {
		// Paths specifies the default list of ignore specifications.
		Paths []string `yaml:"paths" mapstructure:"paths"`
		// VCS specifies the VCS ignore mode.
		VCS core.IgnoreVCSMode `yaml:"vcs" mapstructure:"vcs"`
	} `yaml:"ignore" mapstructure:"ignore"`
	// Symlink contains parameters related to symbolic link handling.
	Symlink struct {
		// Mode specifies the symbolic link mode.
		Mode core.SymbolicLinkMode `yaml:"mode" mapstructure:"mode"`
	} `yaml:"symlink" mapstructure:"symlink"`
	// Watch contains parameters related to filesystem monitoring.
	Watch struct {
		// Mode specifies the file watching mode.
		Mode synchronization.WatchMode `yaml:"mode" mapstructure:"mode"`
		// PollingInterval specifies the interval (in seconds) for poll-based
		// file monitoring. A value of 0 specifies that Mutagen's internal
		// default interval should be used.
		PollingInterval uint32 `yaml:"pollingInterval" mapstructure:"pollingInterval"`
	} `yaml:"watch" mapstructure:"watch"`
	// Permissions contains parameters related to permission handling.
	Permissions struct {
		// DefaultFileMode specifies the default permission mode to use for new
		// files in "portable" permission propagation mode.
		DefaultFileMode filesystem.Mode `yaml:"defaultFileMode" mapstructure:"defaultFileMode"`
		// DefaultDirectoryMode specifies the default permission mode to use for
		// new files in "portable" permission propagation mode.
		DefaultDirectoryMode filesystem.Mode `yaml:"defaultDirectoryMode" mapstructure:"defaultDirectoryMode"`
		// DefaultOwner specifies the default owner identifier to use when
		// setting ownership of new files and directories in "portable"
		// permission propagation mode.
		DefaultOwner string `yaml:"defaultOwner" mapstructure:"defaultOwner"`
		// DefaultGroup specifies the default group identifier to use when
		// setting ownership of new files and directories in "portable"
		// permission propagation mode.
		DefaultGroup string `yaml:"defaultGroup" mapstructure:"defaultGroup"`
	} `yaml:"permissions" mapstructure:"permissions"`
}

// Configuration converts a YAML-based session configuration to a Protocol
// Buffers session configuration. It does not validate the resulting
// configuration.
func (c *Configuration) Configuration() *synchronization.Configuration {
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
