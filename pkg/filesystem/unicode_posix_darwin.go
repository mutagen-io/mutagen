package filesystem

import (
	"golang.org/x/sys/unix"
)

// extractUnicodeDecompositionFromStatfs trys to extract information about
// Unicode decomposition behavior from filesystem metadata by matching the
// filesystem type to one with known behavior.
func extractUnicodeDecompositionFromStatfs(filesystemMetadata *unix.Statfs_t) (bool, bool) {
	// Check if the filesystem is APFS. APFS does not decompose Unicode.
	if statfsRepresentsAPFS(filesystemMetadata) {
		return false, true
	}

	// Check if the filesystem is HFS (or some variant of HFS). HFS does
	// decompose Unicode.
	if statfsRepresentsHFS(filesystemMetadata) {
		return true, true
	}

	// Otherwise this is an unknown filesystem, so we're best off performing
	// full probe operations.
	return false, false
}

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test by path, without probe files. The successfulness of the
// test is indicated by the second return parameter.
func probeUnicodeDecompositionFastByPath(path string) (bool, bool) {
	// Perform a filesystem metadata query on the directory.
	var filesystemMetadata unix.Statfs_t
	if unix.Statfs(path, &filesystemMetadata) != nil {
		return false, false
	}

	// Extract information from the metadata.
	return extractUnicodeDecompositionFromStatfs(&filesystemMetadata)
}

// probeUnicodeDecompositionFast attempts to perform a fast Unicode
// decomposition test, without probe files. The successfulness of the test is
// indicated by the second return parameter.
func probeUnicodeDecompositionFast(directory *Directory) (bool, bool) {
	// Perform a filesystem metadata query on the directory.
	var filesystemMetadata unix.Statfs_t
	if unix.Fstatfs(directory.descriptor, &filesystemMetadata) != nil {
		return false, false
	}

	// Extract information from the metadata.
	return extractUnicodeDecompositionFromStatfs(&filesystemMetadata)
}
