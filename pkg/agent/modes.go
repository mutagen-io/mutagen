package agent

const (
	// ModeInstall is the agent command to invoke for installation.
	ModeInstall = "install"
	// ModeSynchronizer is the agent command to invoke for running as a
	// synchronizer.
	ModeSynchronizer = "synchronizer"
	// ModeForwarder is the agent command to invoke for running as a forwarder.
	ModeForwarder = "forwarder"
	// ModeVersion is the agent command to invoke to print version information.
	ModeVersion = "version"
	// ModeLegal is the agent command to invoke to print legal information.
	ModeLegal = "legal"
)
