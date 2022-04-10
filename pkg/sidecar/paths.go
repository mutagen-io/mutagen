package sidecar

import (
	"path/filepath"
	"strings"
)

// VolumeMountPointForPath returns the mount point for the volume on which path
// resides. If the path does not reside at or beneath a volume mount point, then
// an empty string is returned. The provided path must be absolute, cleaned, and
// fully resolved of symbolic links. This function is only valid in the context
// of a Mutagen sidecar container.
func VolumeMountPointForPath(path string) string {
	// Verify that the path exists at or beneath a volume mount point.
	if !strings.HasPrefix(path, volumeMountParent) || path == volumeMountParent {
		return ""
	}

	// Extract the volume name.
	volume := path[len(volumeMountParent):]
	if index := strings.IndexByte(volume, filepath.Separator); index >= 0 {
		volume = volume[:index]
	}

	// Validate the volume name.
	if volume == "" {
		return ""
	}

	// Compute the volume mount point.
	return volumeMountParent + volume
}
