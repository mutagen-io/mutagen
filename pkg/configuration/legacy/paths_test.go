package legacy

import (
	"testing"
)

// TestConfigurationPath tests that ConfigurationPath succeeds and returns a
// non-empty path.
func TestConfigurationPath(t *testing.T) {
	if path, err := ConfigurationPath(); err != nil {
		t.Fatal("unable to compute legacy global configuration path:", err)
	} else if path == "" {
		t.Error("legacy global configuration path is empty")
	}
}
