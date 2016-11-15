// +build !darwin

package filesystem

func normalizeDirectoryNames(_ string, names []string) error {
	return nil
}
