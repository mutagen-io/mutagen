package filesystem

func isExecutabilityTestPath(_ string) bool {
	return false
}

func PreservesExecutability(_ string) (bool, error) {
	return false, nil
}
