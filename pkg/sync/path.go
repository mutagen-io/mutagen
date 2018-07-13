package sync

// pathJoin is a fast alternative to path.Join that avoids the unnecessary path
// cleaning overhead incurred by that function. It is only for Mutagen's
// in-memory paths, not filesystem paths, for which path/filepath.Join should
// still be used.
func pathJoin(base, leaf string) string {
	// When joining a path to the root, we don't want to concatenate.
	if base == "" {
		return leaf
	}

	// Concatenate the paths.
	return base + "/" + leaf
}
