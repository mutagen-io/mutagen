//go:build !windows

package sidecar

import (
	"path/filepath"
)

const (
	// volumeMountParent is the parent path for all volume mounts within the
	// sidecar container.
	volumeMountParent = "/volumes"
)

// PathIsVolumeMountPoint returns whether or not a path is expected to be a
// volume mount point within a sidecar container. It is only valid in the
// context of a sidecar container.
func PathIsVolumeMountPoint(path string) bool {
	return filepath.Dir(path) == volumeMountParent
}
