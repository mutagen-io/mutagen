package agent

import (
	"os"
	"testing"
)

// TestMain is the entry point for integration tests (overriding the default
// generated entry point).
func TestMain(m *testing.M) {
	// Override the expected bundle location.
	ExpectedBundleLocation = BundleLocationBuildDirectory

	// Run tests.
	os.Exit(m.Run())
}
