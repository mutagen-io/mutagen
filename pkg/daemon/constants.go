package daemon

const (
	// MaximumIPCMessageSize specifies the maximum message size that we'll allow
	// over IPC channels.
	MaximumIPCMessageSize = 25 * 1024 * 1024
)
