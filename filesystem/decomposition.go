package filesystem

// DecomposesUnicode determines whether or not the filesystem at root decomposes
// Unicode characters. It checks if a string in NFC form can be accessed by its
// NFD equivalent. If an error is returned, the determination could not be made
// and its value should be ignored. This test is primarily for OS X with HFS+,
// where Unicode is decomposed into a variant of NFD. If we see similar bad
// behavior on other platforms, we might need to expand this test.
func DecomposesUnicode(root string) (bool, error) {
	return insensitive(root, "mutag\xc3\xa9n_comp", "mutag\x65\xcc\x81n_comp")
}
