//go:build !windows

package sidecar

import (
	"testing"
)

// TestPathIsVolumeMountPoint tests PathIsVolumeMountPoint.
func TestPathIsVolumeMountPoint(t *testing.T) {
	// Define test cases.
	tests := []struct {
		path               string
		expectedVolumeness bool
		expectedVolumeName string
	}{
		{"", false, ""},
		{"/", false, ""},
		{"/volume", false, ""},
		{"/volume/", false, ""},
		{"/volume/fake", false, ""},
		{"/whatever", false, ""},
		{"/whatever ", false, ""},
		{"/whatever /", false, ""},
		{"/volumes/", false, ""},
		{"/volumes/ ", true, " "},
		{"/volumes/name", true, "name"},
		{"/volumes/name/", false, ""},
		{"/volumes/my volume", true, "my volume"},
		{"/volumes/ volume", true, " volume"},
		{"/volumes/name/sub", false, ""},
		{"/volumes/my volume/sub", false, ""},
		{"/volumes/name/sub/second", false, ""},
		{"/volumes/my volume/sub/second sub", false, ""},
	}

	// Process test cases.
	for i, test := range tests {
		volumeness, volumeName := PathIsVolumeMountPoint(test.path)
		if volumeness != test.expectedVolumeness {
			t.Errorf("test case %d: volumeness does not match expected: %t != %t",
				i, volumeness, test.expectedVolumeness,
			)
		}
		if volumeName != test.expectedVolumeName {
			t.Errorf("test case %d: volume name does not match expected: %s != %s",
				i, volumeName, test.expectedVolumeName,
			)
		}
	}
}

// TestVolumeMountPointForPath tests VolumeMountPointForPath.
func TestVolumeMountPointForPath(t *testing.T) {
	// Define test cases.
	tests := []struct {
		path                     string
		expectedVolumeMountPoint string
	}{
		{"", ""},
		{"/", ""},
		{"/volume", ""},
		{"/volume/", ""},
		{"/volume/fake", ""},
		{"/whatever", ""},
		{"/whatever ", ""},
		{"/whatever /", ""},
		{"/volumes/", ""},
		{"/volumes/ ", "/volumes/ "},
		{"/volumes/name", "/volumes/name"},
		{"/volumes/name/", "/volumes/name"},
		{"/volumes/my volume", "/volumes/my volume"},
		{"/volumes/ volume", "/volumes/ volume"},
		{"/volumes/name/sub", "/volumes/name"},
		{"/volumes/my volume/sub", "/volumes/my volume"},
		{"/volumes/name/sub/second", "/volumes/name"},
		{"/volumes/my volume/sub/second sub", "/volumes/my volume"},
	}

	// Process test cases.
	for i, test := range tests {
		volumeMountPoint := VolumeMountPointForPath(test.path)
		if volumeMountPoint != test.expectedVolumeMountPoint {
			t.Errorf("test case %d: volume mount point does not match expected: %s != %s",
				i, volumeMountPoint, test.expectedVolumeMountPoint,
			)
		}
	}
}
