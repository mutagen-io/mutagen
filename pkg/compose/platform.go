package compose

// isSupportedDaemonOS returns whether or not a Docker daemon OS is supported by
// Mutagen's Docker Compose integration.
func isSupportedDaemonOS(daemonOS string) bool {
	switch daemonOS {
	case "linux":
		return true
	case "windows":
		// TODO: Enable once Windows support is implemented.
		return false
	default:
		return false
	}
}
