package configuration

import (
	"testing"
)

// TestLoad tests the Load function.
func TestLoad(t *testing.T) {
	// Attempt to load the global configuration.
	if configuration, err := Load(""); err != nil {
		t.Error("unable to load global configuration:", err)
	} else if configuration == nil {
		t.Error("nil configuration returned")
	}
}

// TODO: Implement additional tests.
