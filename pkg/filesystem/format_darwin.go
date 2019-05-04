package filesystem

import (
	"github.com/pkg/errors"

	"golang.org/x/sys/unix"
)

// volumeFormat represents a filesystem volume format.
type volumeFormat uint8

const (
	// volumeFormatUnknown represents an unknown format.
	volumeFormatUnknown volumeFormat = iota
	// volumeFormatAPFS represents an APFS filesystem format.
	volumeFormatAPFS
	// volumeFormatHFS represents an HFS (or variant thereof) filesystem format.
	volumeFormatHFS
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

// volumeFormatFromMetadata extracts the filesystem format from the filesystem
// metadata.
func volumeFormatFromMetadata(metadata *unix.Statfs_t) volumeFormat {
	// Check if this is a well-known filesystem format.
	if metadataRepresentsAPFS(metadata) {
		return volumeFormatAPFS
	} else if metadataRepresentsHFS(metadata) {
		return volumeFormatHFS
	}

	// Otherwise classify it as unknown.
	return volumeFormatUnknown
}

// queryVolumeFormatByPath queries the filesystem format for the specified path.
func queryVolumeFormatByPath(path string) (volumeFormat, error) {
	// Perform a filesystem metadata query on the path.
	var metadata unix.Statfs_t
	if err := unix.Statfs(path, &metadata); err != nil {
		return volumeFormatUnknown, errors.Wrap(err, "unable to query filesystem metadata")
	}

	// Classify the filesystem.
	return volumeFormatFromMetadata(&metadata), nil
}

// queryVolumeFormat queries the filesystem format for the specified directory.
func queryVolumeFormat(directory *Directory) (volumeFormat, error) {
	// Perform a filesystem metadata query on the directory.
	var metadata unix.Statfs_t
	if err := unix.Fstatfs(directory.descriptor, &metadata); err != nil {
		return volumeFormatUnknown, errors.Wrap(err, "unable to query filesystem metadata")
	}

	// Classify the filesystem.
	return volumeFormatFromMetadata(&metadata), nil
}
