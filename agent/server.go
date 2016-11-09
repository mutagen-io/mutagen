package agent

import (
	"google.golang.org/grpc"
)

func NewServer() *grpc.Server {
	// Create an empty server.
	server := grpc.NewServer()

	// TODO: Register filesystem service.

	// TODO: Register endpoint service.

	// Success.
	return server
}
