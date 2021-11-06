package sidecar

import (
	"path/filepath"
	"strings"
)

const (
	// volumeMountParent is the parent path for all volume mounts within the
	// sidecar container.
	volumeMountParent = `c:\volumes`
)

// PathIsVolumeMountPoint returns whether or not a path is expected to be a
// volume mount point within a sidecar container. It is only valid in the
// context of a sidecar container.
func PathIsVolumeMountPoint(path string) bool {
	// On Windows, we have to perform additional normalization, because the
	// path/filepath.Dir will treat trailing slashes as indicating an empty (but
	// present) leaf directory name. NTFS paths are also case-insensitive.
	return strings.ToLower(filepath.Dir(strings.TrimRight(strings.ReplaceAll(path, "/", "\\"), "\\"))) == volumeMountParent
}
