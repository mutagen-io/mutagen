package global

import (
	"testing"
)

// TestConfigurationPath tests that ConfigurationPath succeeds and returns a
// non-empty path.
func TestConfigurationPath(t *testing.T) {
	if path, err := ConfigurationPath(); err != nil {
		t.Fatal("unable to compute global configuration path:", err)
	} else if path == "" {
		t.Error("global configuration path is empty")
	}
}
