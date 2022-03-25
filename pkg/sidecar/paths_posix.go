//go:build !windows

package sidecar

import (
	"path/filepath"
	"strings"
)

const (
	// volumeMountParent is the parent path for all volume mounts within a
	// Mutagen sidecar container.
	volumeMountParent = "/volumes/"
)

// PathIsVolumeMountPoint returns whether or not a path is expected to be a
// volume mount point within a Mutagen sidecar container. If the path is a
// volume mount point, then the volume name is also returned. The provided path
// must be absolute, cleaned, and fully resolved of symbolic links. This
// function is only valid in the context of a Mutagen sidecar container.
func PathIsVolumeMountPoint(path string) (bool, string) {
	// Parse the path.
	directory, leaf := filepath.Split(path)

	// Verify that the parent directory is the volume mount parent and that the
	// volume name is non-empty.
	if directory == volumeMountParent && leaf != "" {
		return true, leaf
	}
	return false, ""
}

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
