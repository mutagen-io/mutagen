// +build !windows

package sync

func pathContainsInvalidCharacters(path string) bool {
	return false
}
