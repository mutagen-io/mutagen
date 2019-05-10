package format

import (
	"golang.org/x/sys/unix"
)

const (
	// FormatEXT represents an EXT2, EXT3, or EXT4 filesystem format.
	FormatEXT Format = iota + 1
	// FormatNFS represents an NFS filesystem format.
	FormatNFS
)

// formatFromStatfs extracts the filesystem format from the filesystem metadata.
func formatFromStatfs(metadata *unix.Statfs_t) Format {
	switch metadata.Type {
	case unix.EXT4_SUPER_MAGIC:
		return FormatEXT
	case unix.NFS_SUPER_MAGIC:
		return FormatNFS
	default:
		return FormatUnknown
	}
}
