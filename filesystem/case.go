package filesystem

// CaseInsensitive determines whether or not the filesystem at root is case
// insensitive. It says nothing about case preservation, but that is provided by
// all modern filesystems. If an error is returned, the determination could not
// be made and its value should be ignored.
func CaseInsensitive(root string) (bool, error) {
	return insensitive(root, "mutagen_case", "MUTAGEN_case")
}
