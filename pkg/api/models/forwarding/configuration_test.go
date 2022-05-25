package forwarding

import (
	"os"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
)

const (
	testYAMLConfiguration = `
socket:
  overwriteMode: "overwrite"
  owner: "george"
  group: "presidents"
  permissionMode: 0600
`
)

// expectedConfiguration is the configuration that's expected based on the
// human-readable configuration given above.
var expectedConfiguration = &forwarding.Configuration{
	SocketOverwriteMode:  forwarding.SocketOverwriteMode_SocketOverwriteModeOverwrite,
	SocketOwner:          "george",
	SocketGroup:          "presidents",
	SocketPermissionMode: 0600,
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
	if configuration.SocketOverwriteMode != expectedConfiguration.SocketOverwriteMode {
		t.Error("socket overwrite mode mismatch:", configuration.SocketOverwriteMode, "!=", expectedConfiguration.SocketOverwriteMode)
	}
	if configuration.SocketOwner != expectedConfiguration.SocketOwner {
		t.Error("socket owner mismatch:", configuration.SocketOwner, "!=", expectedConfiguration.SocketOwner)
	}
	if configuration.SocketGroup != expectedConfiguration.SocketGroup {
		t.Error("socket owner mismatch:", configuration.SocketGroup, "!=", expectedConfiguration.SocketGroup)
	}
	if configuration.SocketPermissionMode != expectedConfiguration.SocketPermissionMode {
		t.Errorf("socket permission mode mismatch: %o != %o", configuration.SocketPermissionMode, expectedConfiguration.SocketPermissionMode)
	}
}

// TODO: Expand tests, including testing for invalid configurations.
