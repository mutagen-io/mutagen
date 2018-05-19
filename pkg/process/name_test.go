package process

import (
	"testing"
)

func TestExecutableNameWindows(t *testing.T) {
	if name := ExecutableName("mutagen-agent", "windows"); name != "mutagen-agent.exe" {
		t.Error("executable name incorrect for Windows")
	}
}

func TestExecutableNameLinux(t *testing.T) {
	if name := ExecutableName("mutagen-agent", "linux"); name != "mutagen-agent" {
		t.Error("executable name incorrect for Linux")
	}
}
