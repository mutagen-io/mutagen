package sync

import (
	"strings"
)

// pathJoin is a fast alternative to path.Join that avoids the unnecessary path
// cleaning overhead incurred by that function. It is only for Mutagen's
// in-memory paths, not filesystem paths, for which path/filepath.Join should
// still be used. The provided leaf name must be non-empty, otherwise this
// function will panic.
func pathJoin(base, leaf string) string {
	// Disalllow empty leaf names.
	if leaf == "" {
		panic("empty leaf name")
	}

	// When joining a path to the synchronization root, we don't want to
	// concatenate.
	if base == "" {
		return leaf
	}

	// Concatenate the paths.
	return base + "/" + leaf
}

// pathDir is a fast alternative to path.Dir that avoids the unnecessary path
// cleaning overhead incurred by that function. It is only for Mutagen's
// in-memory paths, not filesystem paths, for which path/filepath.Dir should
// still be used. Note that this function isn't equivalent to returning the
// first return value from path.Split, because in that case the trailing slash
// remains in the directory path. The provided path must be non-empty, otherwise
// this function will panic.
func pathDir(path string) string {
	// Disallow synchronization root paths.
	if path == "" {
		panic("empty path")
	}

	// Identify the index of the last slash in the path.
	lastSlashIndex := strings.LastIndexByte(path, '/')

	// If there is no slash, then the parent is the synchronization root.
	if lastSlashIndex == -1 {
		return ""
	}

	// Verify that the parent path isn't empty. There aren't any scenarios where
	// this is allowed.
	if lastSlashIndex == 0 {
		panic("empty parent path")
	}

	// Trim off the slash and everything that follows.
	return path[:lastSlashIndex]
}
