package filesystem

// IsExecutabilityTestFileName determines whether or not a file name (not a file
// path) is the name of an executability preservation probe file. On Windows
// this function always returns false since probe files are not used.
func IsExecutabilityTestFileName(_ string) bool {
	return false
}

// PreservesExecutability determines whether or not the directory at the
// specified path preserves POSIX executability bits. On Windows this function
// always returns false since POSIX executability bits are never preserved.
func PreservesExecutability(_ string) (bool, error) {
	return false, nil
}
