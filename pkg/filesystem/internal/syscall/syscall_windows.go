package syscall

import (
	"golang.org/x/sys/windows"
)

// FileAttributeTagInfo is the Go representation of FILE_ATTRIBUTE_TAG_INFO.
type FileAttributeTagInfo struct {
	// FileAttributes are the file attributes.
	FileAttributes uint32
	// ReparseTag is the file reparse tag.
	ReparseTag uint32
}

// IsSymbolicLink returns whether or not the file attributes indicate a symbolic
// link. This method must match the implementation of os.fileStat.isSymlink in
// the Go standard library in order for code in the filesystem package to work
// correctly (since it partially relies on os package functionality on Windows
// and will see inconsistent classification behavior otherwise).
func (i *FileAttributeTagInfo) IsSymbolicLink() bool {
	if i.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
		return false
	}
	// TODO: Update this definition once golang/go#42184 is resolved.
	return i.ReparseTag == windows.IO_REPARSE_TAG_SYMLINK ||
		i.ReparseTag == windows.IO_REPARSE_TAG_MOUNT_POINT
}
