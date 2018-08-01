package process

import (
	"testing"
)

// TestExecutableNameWindows tests that ExecutableName works correctly for a
// Windows target.
func TestExecutableNameWindows(t *testing.T) {
	if name := ExecutableName("mutagen-agent", "windows"); name != "mutagen-agent.exe" {
		t.Error("executable name incorrect for Windows")
	}
}

// TestExecutableNameLinux tests that ExecutableName works correctly for a Linux
// target.
func TestExecutableNameLinux(t *testing.T) {
	if name := ExecutableName("mutagen-agent", "linux"); name != "mutagen-agent" {
		t.Error("executable name incorrect for Linux")
	}
}
