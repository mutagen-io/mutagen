package session

import (
	"net"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/connectivity"
	"github.com/havoc-io/mutagen/grpcutil"
)

func dialLocal() *grpc.ClientConn {
	// Create a gRPC server.
	server := grpc.NewServer()

	// Register an endpoint service.
	RegisterEndpointServer(server, NewEndpoint())

	// Create an in-memory pipe.
	clientConnection, serverConnection := net.Pipe()

	// Create a one-shot listener and start serving on that listener. This
	// listener will error out after the first accept, but by that time the lone
	// pipe connection will have been accepted and its processing will have
	// started in a separate Goroutine (where the server will live on). This
	// Goroutine will exit when the connection closes.
	listener := connectivity.NewOneShotListener(serverConnection)
	server.Serve(listener)

	// Create a gRPC client using this connection.
	return grpcutil.NewNonRedialingClientConnection(clientConnection)
}
