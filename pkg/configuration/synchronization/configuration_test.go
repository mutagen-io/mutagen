package synchronization

import (
	"os"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/filesystem/behavior"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

const (
	testYAMLConfiguration = `
mode: "two-way-resolved"
maxEntryCount: 500
maxStagingFileSize: "1000 GB"
probeMode: "assume"
scanMode: "accelerated"
stageMode: "neighboring"

symlink:
  mode: "portable"

watch:
  mode: "force-poll"
  pollingInterval: 5

ignore:
  paths:
    - "ignore/this/**"
    - "!ignore/this/that"
  vcs: true

permissions:
  defaultFileMode: 644
  defaultDirectoryMode: 0755
  defaultOwner: "george"
  defaultGroup: "presidents"
`
)

// expectedConfiguration is the configuration that's expected based on the
// human-readable configuration given above.
var expectedConfiguration = &synchronization.Configuration{
	SynchronizationMode: core.SynchronizationMode_SynchronizationModeTwoWayResolved,
	MaximumEntryCount:   500,
	// TODO: This will mis-match.
	MaximumStagingFileSize: 1000000000000,
	ProbeMode:              behavior.ProbeMode_ProbeModeAssume,
	ScanMode:               synchronization.ScanMode_ScanModeAccelerated,
	StageMode:              synchronization.StageMode_StageModeNeighboring,
	SymlinkMode:            core.SymlinkMode_SymlinkModePortable,
	WatchMode:              synchronization.WatchMode_WatchModeForcePoll,
	WatchPollingInterval:   5,
	Ignores: []string{
		"ignore/this/**",
		"!ignore/this/that",
	},
	IgnoreVCSMode:        core.IgnoreVCSMode_IgnoreVCSModeIgnore,
	DefaultFileMode:      0644,
	DefaultDirectoryMode: 0755,
	DefaultOwner:         "george",
	DefaultGroup:         "presidents",
}

// TestLoadConfiguration tests loading a YAML-based session configuration.
func TestLoadConfiguration(t *testing.T) {
	// Write a valid configuration to a temporary file and defer its cleanup.
	file, err := os.CreateTemp("", "mutagen_configuration")
	if err != nil {
		t.Fatal("unable to create temporary file:", err)
	} else if _, err = file.Write([]byte(testYAMLConfiguration)); err != nil {
		t.Fatal("unable to write data to temporary file:", err)
	} else if err = file.Close(); err != nil {
		t.Fatal("unable to close temporary file:", err)
	}
	defer os.Remove(file.Name())

	// Attempt to load.
	yamlConfiguration := &Configuration{}
	if err := encoding.LoadAndUnmarshalYAML(file.Name(), yamlConfiguration); err != nil {
		t.Error("configuration loading failed:", err)
	}

	// Compute the Protocol Buffers session representation.
	configuration := yamlConfiguration.Configuration()

	// Ensure that the resulting configuration is valid.
	if err := configuration.EnsureValid(false); err != nil {
		t.Error("derived configuration invalid:", err)
	}

	// Verify that the configuration matches what's expected.
	if configuration.SynchronizationMode != expectedConfiguration.SynchronizationMode {
		t.Error("synchronization mode mismatch:", configuration.SynchronizationMode, "!=", expectedConfiguration.SynchronizationMode)
	}
	if configuration.MaximumEntryCount != expectedConfiguration.MaximumEntryCount {
		t.Error("maximum entry count mismatch:", configuration.MaximumEntryCount, "!=", expectedConfiguration.MaximumEntryCount)
	}
	if configuration.MaximumStagingFileSize != expectedConfiguration.MaximumStagingFileSize {
		t.Error("maximum staging file size mismatch:", configuration.MaximumStagingFileSize, "!=", expectedConfiguration.MaximumStagingFileSize)
	}
	if configuration.ProbeMode != expectedConfiguration.ProbeMode {
		t.Error("probe mode mismatch:", configuration.ProbeMode, "!=", expectedConfiguration.ProbeMode)
	}
	if configuration.ScanMode != expectedConfiguration.ScanMode {
		t.Error("scan mode mismatch:", configuration.ScanMode, "!=", expectedConfiguration.ScanMode)
	}
	if configuration.StageMode != expectedConfiguration.StageMode {
		t.Error("stage mode mismatch:", configuration.StageMode, "!=", expectedConfiguration.StageMode)
	}
	if configuration.SymlinkMode != expectedConfiguration.SymlinkMode {
		t.Error("symlink mode mismatch:", configuration.SymlinkMode, "!=", expectedConfiguration.SymlinkMode)
	}
	if configuration.WatchMode != expectedConfiguration.WatchMode {
		t.Error("watch mode mismatch:", configuration.WatchMode, "!=", expectedConfiguration.WatchMode)
	}
	if configuration.WatchPollingInterval != expectedConfiguration.WatchPollingInterval {
		t.Error("watch polling interval mismatch:", configuration.WatchPollingInterval, "!=", expectedConfiguration.WatchPollingInterval)
	}
	if len(configuration.Ignores) != len(expectedConfiguration.Ignores) {
		t.Error("ignore count mismatch:", len(configuration.Ignores), "!=", len(expectedConfiguration.Ignores))
	} else {
		for i, ignore := range configuration.Ignores {
			if ignore != expectedConfiguration.Ignores[i] {
				t.Error("ignore mismatch:", ignore, "!=", expectedConfiguration.Ignores[i], "at index", i)
			}
		}
	}
	if configuration.IgnoreVCSMode != expectedConfiguration.IgnoreVCSMode {
		t.Error("ignore VCS mode mismatch:", configuration.IgnoreVCSMode, "!=", expectedConfiguration.IgnoreVCSMode)
	}
	if configuration.DefaultFileMode != expectedConfiguration.DefaultFileMode {
		t.Errorf("default file mode mismatch: %o != %o", configuration.DefaultFileMode, expectedConfiguration.DefaultFileMode)
	}
	if configuration.DefaultDirectoryMode != expectedConfiguration.DefaultDirectoryMode {
		t.Errorf("default directory mode mismatch: %o != %o", configuration.DefaultDirectoryMode, expectedConfiguration.DefaultDirectoryMode)
	}
	if configuration.DefaultOwner != expectedConfiguration.DefaultOwner {
		t.Error("default owner mismatch:", configuration.DefaultOwner, "!=", expectedConfiguration.DefaultOwner)
	}
	if configuration.DefaultGroup != expectedConfiguration.DefaultGroup {
		t.Error("default owner mismatch:", configuration.DefaultGroup, "!=", expectedConfiguration.DefaultGroup)
	}
}

// TODO: Expand tests, including testing for invalid configurations.
