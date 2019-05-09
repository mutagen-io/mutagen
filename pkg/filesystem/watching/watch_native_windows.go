package watching

import (
	"os"
	"syscall"
)

// watchRootParametersEqual determines whether or not the metadata for a path
// being used as a watch root has changed sufficiently to warrant recreating the
// watch.
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
