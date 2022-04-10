//go:build !windows

package sidecar

import (
	"testing"
)

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
