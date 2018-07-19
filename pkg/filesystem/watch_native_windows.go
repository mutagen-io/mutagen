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
	firstData, firstOk := first.Sys().(*syscall.Win32FileAttributeData)
	secondData, secondOk := second.Sys().(*syscall.Win32FileAttributeData)

	// Check for equality.
	return firstOk && secondOk &&
		firstData.FileAttributes == secondData.FileAttributes &&
		firstData.CreationTime == secondData.CreationTime
}
