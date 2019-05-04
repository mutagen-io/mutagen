package filesystem

import (
	"golang.org/x/sys/unix"
)

// extractExecutabilityPreservationFromStatfs trys to extract information about
// executability preservation behavior from filesystem metadata by matching the
// filesystem type to one with known behavior.
func extractExecutabilityPreservationFromStatfs(filesystemMetadata *unix.Statfs_t) (bool, bool) {
	// Check if the filesystem is APFS. APFS does preserve executability.
	if statfsRepresentsAPFS(filesystemMetadata) {
		return true, true
	}

	// Check if the filesystem is HFS (or some variant of HFS). HFS does
	// preserve executability.
	if statfsRepresentsHFS(filesystemMetadata) {
		return true, true
	}

	// Otherwise this is an unknown filesystem, so we're best off performing
	// full probe operations.
	return false, false
}

// probeExecutabilityPreservationFastByPath attempts to perform a fast
// executability preservation test by path, without probe files. The
// successfulness of the test is indicated by the second return parameter.
func probeExecutabilityPreservationFastByPath(path string) (bool, bool) {
	// Perform a filesystem metadata query on the directory.
	var filesystemMetadata unix.Statfs_t
	if unix.Statfs(path, &filesystemMetadata) != nil {
		return false, false
	}

	// Extract information from the metadata.
	return extractExecutabilityPreservationFromStatfs(&filesystemMetadata)
}

// probeExecutabilityPreservationFast attempts to perform a fast executability
// preservation test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeExecutabilityPreservationFast(directory *Directory) (bool, bool) {
	// Perform a filesystem metadata query on the directory.
	var filesystemMetadata unix.Statfs_t
	if unix.Fstatfs(directory.descriptor, &filesystemMetadata) != nil {
		return false, false
	}

	// Extract information from the metadata.
	return extractExecutabilityPreservationFromStatfs(&filesystemMetadata)
}
