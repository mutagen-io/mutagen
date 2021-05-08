package daemon

const (
	// Host is the hostname on which the daemon listens.
	Host = "localhost"
	// DefaultPort is the default TCP port used by the daemon. This is not
	// necessarily the port being used by the daemon, which may instead be
	// listening on a port (potentially a dynamic one) specified via an
	// environment variable. Clients should always read the daemon port from
	// disk or use the mutagen daemon port command to retrieve the daemon port.
	DefaultPort = 31116
)
