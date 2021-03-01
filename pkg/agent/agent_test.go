package agent

import (
	"testing"
)

// TestMain is the entry point for integration tests. It replaces the default
// test entry point so that it can override the agent bundle location.
func TestMain(m *testing.M) {
	// Override the expected bundle location.
	ExpectedBundleLocation = BundleLocationBuildDirectory

	// Run tests.
	m.Run()
}
