package url

// isWindowsPath determines whether or not a raw URL string is a Windows path.
func isWindowsPath(raw string) bool {
	// These will all be single-byte runes, so we can do direct byte access and
	// comparison.
	return len(raw) >= 3 &&
		((raw[0] >= 'a' && raw[0] <= 'z') || (raw[0] >= 'A' && raw[0] <= 'Z')) &&
		raw[1] == ':' &&
		(raw[2] == '\\' || raw[2] == '/')
}
