package core

import (
	"os"
)

// testingFilesystem encodes information about a test filesystem.
type testingFilesystem struct {
	// name is a human-readable name describing the filesystem.
	name string
	// storage is the temporary directory storage path of the filesystem. It
	// should be suitable for use as the directory argument of os.MkdirTemp and
	// thus may be empty to represent the system temporary directory.
	storage string
}

// testingFilesystems are the filesystems available for testing.
var testingFilesystems []testingFilesystem

func init() {
	// Add the default system temporary directory for testing.
	testingFilesystems = append(testingFilesystems, testingFilesystem{"OS", ""})

	// If there's a FAT32 root to test in, then add that.
	if root := os.Getenv("MUTAGEN_TEST_FAT32_ROOT"); root != "" {
		testingFilesystems = append(testingFilesystems, testingFilesystem{"FAT32", root})
	}

	// If there's an HFS+ root to test in, then add that.
	if root := os.Getenv("MUTAGEN_TEST_HFS_ROOT"); root != "" {
		testingFilesystems = append(testingFilesystems, testingFilesystem{"HFS+", root})
	}

	// If there's an APFS root to test in, then add that.
	if root := os.Getenv("MUTAGEN_TEST_APFS_ROOT"); root != "" {
		testingFilesystems = append(testingFilesystems, testingFilesystem{"APFS", root})
	}
}
