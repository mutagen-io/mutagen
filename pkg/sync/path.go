package sync

// pathJoin is a fast alternative to path.Join that avoids the unnecessary path
// cleaning overhead incurred by that function. It is only for Mutagen's
// in-memory paths, not filesystem paths, for which path/filepath.Join should
// still be used.
func pathJoin(base, leaf string) string {
	return base + "/" + leaf
}
