package filesystem

// IsUnicodeProbeFileName determines whether or not a file name (not a file
// path) is the name of an Unicode decomposition probe file. On Windows this
// function always returns false since Unicode probe files are not used.
func IsUnicodeProbeFileName(_ string) bool {
	return false
}

// DecomposesUnicode determines whether or not the filesystem on which the
// directory at the specified path resides decomposes Unicode filenames. On
// Windows this function always returns false since Windows filesystems preserve
// Unicode filename normalization.
func DecomposesUnicode(_ string) (bool, error) {
	return false, nil
}
