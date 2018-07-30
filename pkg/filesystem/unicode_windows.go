package filesystem

func IsUnicodeTestFileName(_ string) bool {
	return false
}

func DecomposesUnicode(_ string) (bool, error) {
	return false, nil
}
