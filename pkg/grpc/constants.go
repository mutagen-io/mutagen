package grpc

const (
	// MaximumIPCMessageSize specifies the maximum message size that we'll allow
	// over IPC channels with gRPC.
	MaximumIPCMessageSize = 25 * 1024 * 1024
)
