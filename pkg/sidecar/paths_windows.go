package sidecar

import (
	"path/filepath"
	"strings"
)

const (
	// volumeMountParent is the parent path for all volume mounts within the
	// sidecar container.
	volumeMountParent = `c:\volumes\`
)

// PathIsVolumeMountPoint returns whether or not a path is expected to be a
// volume mount point within a Mutagen sidecar container. If the path is a
// volume mount point, then the volume name is also returned. This function is
// only valid in the context of a Mutagen sidecar container.
func PathIsVolumeMountPoint(path string) (bool, string) {
	directory, leaf := filepath.Split(filepath.Clean(path))
	if strings.EqualFold(directory, volumeMountParent) {
		return true, leaf
	}
	return false, ""
}
