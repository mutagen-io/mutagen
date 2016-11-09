package daemon

import (
	"google.golang.org/grpc"
)

func NewServer() (*grpc.Server, chan struct{}) {
	// Create an empty server.
	server := grpc.NewServer()

	// Create and register the daemon service.
	daemonService, termination := newService()
	RegisterDaemonServer(server, daemonService)

	// TODO: Create and register the agent service.

	// TODO: Create and register the session service.

	// Success.
	return server, termination
}
