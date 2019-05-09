package filesystem

import (
	"golang.org/x/sys/unix"
)

const (
	// FormatEXT represents an EXT2, EXT3, or EXT4 filesystem format.
	FormatEXT Format = iota + 1
	// FormatNFS represents an NFS filesystem format.
	FormatNFS
)

// formatFromMetadata extracts the filesystem format from the filesystem
// metadata.
func formatFromMetadata(metadata *unix.Statfs_t) Format {
	switch metadata.Type {
	case unix.EXT4_SUPER_MAGIC:
		return FormatEXT
	case unix.NFS_SUPER_MAGIC:
		return FormatNFS
	default:
		return FormatUnknown
	}
}
