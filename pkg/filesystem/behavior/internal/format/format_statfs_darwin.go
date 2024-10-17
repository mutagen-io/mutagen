package format

import (
	"golang.org/x/sys/unix"
)

const (
	// FormatAPFS represents an APFS filesystem format.
	FormatAPFS Format = iota + 1
	// FormatHFS represents an HFS (or variant thereof) filesystem format.
	FormatHFS
	// FormatFAT32 represents a FAT32 filesystem format.
	FormatFAT32
	// FormatExFAT represents a ExFAT filesystem format.
	FormatExFAT
)

// metadataRepresentsAPFS returns whether or not the specified filesystem
// metadata represents an APFS filesystem.
func metadataRepresentsAPFS(metadata *unix.Statfs_t) bool {
	return metadata.Fstypename[0] == 'a' &&
		metadata.Fstypename[1] == 'p' &&
		metadata.Fstypename[2] == 'f' &&
		metadata.Fstypename[3] == 's'
}

// metadataRepresentsHFS returns whether or not the specified filesystem
// metadata represents an HFS filesystem. This also covers HFS variants.
func metadataRepresentsHFS(metadata *unix.Statfs_t) bool {
	return metadata.Fstypename[0] == 'h' &&
		metadata.Fstypename[1] == 'f' &&
		metadata.Fstypename[2] == 's'
}

// metadataRepresentsFAT32 returns whether or not the specified filesystem
// metadata represents a FAT32 filesystem.
func metadataRepresentsFAT32(metadata *unix.Statfs_t) bool {
	return metadata.Fstypename[0] == 'm' &&
		metadata.Fstypename[1] == 's' &&
		metadata.Fstypename[2] == 'd' &&
		metadata.Fstypename[3] == 'o' &&
		metadata.Fstypename[4] == 's'
}

// metadataRepresentsExFAT returns whether or not the specified filesystem
// metadata represents a ExFAT filesystem.
func metadataRepresentsExFAT(metadata *unix.Statfs_t) bool {
	return metadata.Fstypename[0] == 'e' &&
		metadata.Fstypename[1] == 'x' &&
		metadata.Fstypename[2] == 'f' &&
		metadata.Fstypename[3] == 'a' &&
		metadata.Fstypename[4] == 't'
}

// formatFromStatfs extracts the filesystem format from the filesystem metadata.
func formatFromStatfs(metadata *unix.Statfs_t) Format {
	// Check if this is a well-known filesystem format.
	if metadataRepresentsAPFS(metadata) {
		return FormatAPFS
	} else if metadataRepresentsHFS(metadata) {
		return FormatHFS
	} else if metadataRepresentsFAT32(metadata) {
		return FormatFAT32
	} else if metadataRepresentsExFAT(metadata) {
		return FormatExFAT
	}

	// Otherwise classify it as unknown.
	return FormatUnknown
}
