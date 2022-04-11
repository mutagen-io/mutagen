//go:build !windows

package sidecar

// volumeMountParent is the parent path for all volume mounts within the sidecar
// container.
const volumeMountParent = "/volumes/"
