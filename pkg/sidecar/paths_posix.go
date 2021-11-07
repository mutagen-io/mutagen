//go:build !windows

package sidecar

import (
	"path/filepath"
)

const (
	// volumeMountParent is the parent path for all volume mounts within a
	// Mutagen sidecar container.
	volumeMountParent = "/volumes/"
)

// PathIsVolumeMountPoint returns whether or not a path is expected to be a
// volume mount point within a Mutagen sidecar container. If the path is a
// volume mount point, then the volume name is also returned. This function is
// only valid in the context of a Mutagen sidecar container.
func PathIsVolumeMountPoint(path string) (bool, string) {
	directory, leaf := filepath.Split(filepath.Clean(path))
	if directory == volumeMountParent {
		return true, leaf
	}
	return false, ""
}
