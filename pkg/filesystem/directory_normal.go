// +build !darwin

package filesystem

func normalizeDirectoryNames(_ string, _ []string) error {
	return nil
}
