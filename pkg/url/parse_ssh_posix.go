// +build !windows

package url

// isWindowsPath determines whether or not a raw URL string is a Windows path.
func isWindowsPath(raw string) bool {
	return false
}
