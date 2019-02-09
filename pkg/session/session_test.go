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

// TestDefaultFilePermissionModeValid verifies that DefaultFilePermissionMode
// results are valid for use in "portable" permission propagation.
func TestDefaultFilePermissionModeValid(t *testing.T) {
	for _, version := range supportedSessionVersions {
		if err := sync.EnsureDefaultFileModeValid(version.DefaultFilePermissionMode()); err != nil {
			t.Error("invalid default file permission mode:", err)
		}
	}
}

// TestDefaultDirectoryPermissionModeValid verifies that
// DefaultDirectoryPermissionMode results are valid for use in "portable"
// permission propagation.
func TestDefaultDirectoryPermissionModeValid(t *testing.T) {
	for _, version := range supportedSessionVersions {
		if err := sync.EnsureDefaultDirectoryModeValid(version.DefaultDirectoryPermissionMode()); err != nil {
			t.Error("invalid default directory permission mode:", err)
		}
	}
}

// TODO: Implement additional tests.
