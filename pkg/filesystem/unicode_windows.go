package filesystem

func isDecompositionTestPath(_ string) bool {
	return false
}

func DecomposesUnicode(_ string) (bool, error) {
	return false, nil
}
