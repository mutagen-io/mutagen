//go:build !linux && !windows

package filesystem

// extraOpenFlags specifies platform-specific flags to include in open calls.
const extraOpenFlags = 0
