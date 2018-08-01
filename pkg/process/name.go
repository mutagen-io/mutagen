package process

// ExecutableName computes the name for an executable for a given base name on a
// specified operating system.
func ExecutableName(base, goos string) string {
	// If we're on Windows, append ".exe".
	if goos == "windows" {
		return base + ".exe"
	}

	// Otherwise return the base name unmodified.
	return base
}
