package compose

// isSupportedPlatform returns whether or not a Docker platform (i.e. OS) is
// supported by Mutagen's Docker Compose integration.
// TODO: Add support for Windows.
func isSupportedPlatform(platform string) bool {
	switch platform {
	case "linux":
		return true
	default:
		return false
	}
}
