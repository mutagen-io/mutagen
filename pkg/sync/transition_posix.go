// +build !windows

package sync

func containsAlternatePathSeparator(_ string) bool {
	return false
}
