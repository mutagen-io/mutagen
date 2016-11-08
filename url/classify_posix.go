// +build !windows

package url

func isWindowsPath(raw string) bool {
	return false
}
