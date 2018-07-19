// +build !windows,!plan9

package filesystem

import (
	"os"
	"syscall"
)

func watchRootParametersEqual(first, second os.FileInfo) bool {
	// Watch out for nil file information.
	if first == nil && second == nil {
		return true
	} else if first == nil || second == nil {
		return false
	}

	// Extract the underlying metadata.
	firstData, firstOk := first.Sys().(*syscall.Stat_t)
	secondData, secondOk := second.Sys().(*syscall.Stat_t)

	// Check for equality.
	return firstOk && secondOk &&
		firstData.Dev == secondData.Dev &&
		firstData.Ino == secondData.Ino
}
