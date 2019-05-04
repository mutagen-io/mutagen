package filesystem

import (
	"golang.org/x/sys/unix"
)

// statfsRepresentsAPFS returns whether or not the specified filesystem metadata
// represents an APFS filesystem.
func statfsRepresentsAPFS(statfs *unix.Statfs_t) bool {
	return statfs.Fstypename[0] == 'a' &&
		statfs.Fstypename[1] == 'p' &&
		statfs.Fstypename[2] == 'f' &&
		statfs.Fstypename[3] == 's'
}

// statfsRepresentsHFS returns whether or not the specified filesystem metadata
// represents an HFS filesystem. This also covers HFS variants.
func statfsRepresentsHFS(statfs *unix.Statfs_t) bool {
	return statfs.Fstypename[0] == 'h' &&
		statfs.Fstypename[1] == 'f' &&
		statfs.Fstypename[2] == 's'
}
