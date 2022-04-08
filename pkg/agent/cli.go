package agent

const (
	// CommandInstall is the name of the agent installation command.
	CommandInstall = "install"
	// CommandForwarder is the name of the agent forwarder command.
	CommandForwarder = "forwarder"
	// CommandSynchronizer is the name of the agent synchronizer command.
	CommandSynchronizer = "synchronizer"

	// FlagLogLevel is the flag for specifying the log level for the forwarder
	// and synchronizer commands (without the preceding double-dash).
	FlagLogLevel = "log-level"
)
