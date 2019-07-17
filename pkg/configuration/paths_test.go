package configuration

import (
	"testing"
)

// TestGlobalConfigurationPath tests that GlobalConfigurationPath succeeds and
// returns a non-empty path.
func TestGlobalConfigurationPath(t *testing.T) {
	if path, err := GlobalConfigurationPath(); err != nil {
		t.Fatal("unable to compute global configuration path:", err)
	} else if path == "" {
		t.Error("global configuration path is empty")
	}
}
