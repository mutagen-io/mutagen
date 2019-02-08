package filesystem

// IsExecutabilityProbeFileName determines whether or not a file name (not a
// file path) is the name of an executability preservation probe file. On
// Windows this function always returns false since executability probe files
// are not used.
func IsExecutabilityProbeFileName(_ string) bool {
	return false
}

// PreservesExecutabilityByPath determines whether or not the directory at the
// specified path preserves POSIX executability bits. On Windows this function
// always returns false since POSIX executability bits are never preserved.
func PreservesExecutabilityByPath(_ string) (bool, error) {
	return false, nil
}

// PreservesExecutability determines whether or not the specified directory (and
// its underlying filesystem) preserves POSIX executability bits. On Windows
// this function always returns false since POSIX executability bits are never
// preserved.
func PreservesExecutability(_ *Directory) (bool, error) {
	return false, nil
}
