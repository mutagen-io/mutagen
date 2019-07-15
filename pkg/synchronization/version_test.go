package synchronization

import (
	"testing"

	"github.com/mutagen-io/mutagen/pkg/synchronization/core"
)

// supportedSessionVersions defines the supported session versions that should
// be used in testing. It should be updated as new versions are added.
var supportedSessionVersions = []Version{
	Version_Version1,
}

// TestSupportedVersions verifies that session version support is as expected.
func TestSupportedVersions(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		version  Version
		expected bool
	}{
		{Version_Invalid, false},
		{Version_Version1, true},
		{Version_Version1 + 1, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if supported := testCase.version.Supported(); supported != testCase.expected {
			t.Errorf(
				"session version (%s) support does not match expected: %t != %t",
				testCase.version,
				supported,
				testCase.expected,
			)
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
		if err := core.EnsureDefaultFileModeValid(version.DefaultFileMode()); err != nil {
			t.Error("invalid default file mode:", err)
		}
	}
}

// TestDefaultDirectoryModeValid verifies that DefaultDirectoryMode results are
// valid for use in "portable" permission propagation.
func TestDefaultDirectoryModeValid(t *testing.T) {
	for _, version := range supportedSessionVersions {
		if err := core.EnsureDefaultDirectoryModeValid(version.DefaultDirectoryMode()); err != nil {
			t.Error("invalid default directory mode:", err)
		}
	}
}

// TODO: Implement additional tests.
