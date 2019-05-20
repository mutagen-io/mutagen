package session

import (
	"testing"

	"github.com/havoc-io/mutagen/pkg/sync"
)

// supportedSessionVersions defines the supported session versions that should
// be used in testing. It should be updated as new versions are added.
var supportedSessionVersions = []Version{
	Version_Version1,
}

// TestSupportedVersions verifies that all versions that should be supported are
// reported as supported.
func TestSupportedVersions(t *testing.T) {
	for _, version := range supportedSessionVersions {
		if !version.Supported() {
			t.Error("session version reported as unsupported:", version)
		}
	}
}

// TestDefaultWatchPollingIntervalNonZero verifies that
// DefaultWatchPollingInterval results are non-zero, which is required for watch
// operations.
func TestDefaultWatchPollingIntervalNonZero(t *testing.T) {
	for _, version := range supportedSessionVersions {
		if version.DefaultWatchPollingInterval() == 0 {
			t.Error("zero-valued default watch polling interval")
		}
	}
}

// TestDefaultFileModeValid verifies that DefaultFileMode results are valid for
// use in "portable" permission propagation.
func TestDefaultFileModeValid(t *testing.T) {
	for _, version := range supportedSessionVersions {
		if err := sync.EnsureDefaultFileModeValid(version.DefaultFileMode()); err != nil {
			t.Error("invalid default file mode:", err)
		}
	}
}

// TestDefaultDirectoryModeValid verifies that DefaultDirectoryMode results are
// valid for use in "portable" permission propagation.
func TestDefaultDirectoryModeValid(t *testing.T) {
	for _, version := range supportedSessionVersions {
		if err := sync.EnsureDefaultDirectoryModeValid(version.DefaultDirectoryMode()); err != nil {
			t.Error("invalid default directory mode:", err)
		}
	}
}

// TODO: Implement additional tests.
