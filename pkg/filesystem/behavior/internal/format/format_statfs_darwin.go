package format

import (
	"golang.org/x/sys/unix"
)

const (
	// FormatAPFS represents an APFS filesystem format.
	FormatAPFS Format = iota + 1
	// FormatHFS represents an HFS (or variant thereof) filesystem format.
	FormatHFS
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

// formatFromStatfs extracts the filesystem format from the filesystem metadata.
func formatFromStatfs(metadata *unix.Statfs_t) Format {
	// Check if this is a well-known filesystem format.
	if metadataRepresentsAPFS(metadata) {
		return FormatAPFS
	} else if metadataRepresentsHFS(metadata) {
		return FormatHFS
	}

	// Otherwise classify it as unknown.
	return FormatUnknown
}
