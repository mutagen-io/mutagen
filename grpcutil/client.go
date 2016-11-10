package grpcutil

import (
	"net"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc"
)

type oneShotDialerError struct{}

func (e *oneShotDialerError) Error() string {
	return "dialer is one-shot"
}

func (e *oneShotDialerError) Temporary() bool {
	return false
}

// NewNonRedialingClientConnection creates a new grpc.ClientConn that uses the specified
// connection and won't attempt redialing of any sort (or rather, will fail
// permanently when it does). It will specify the WithInsecure dial option, so
// the connection should already be authenticated and secured.
// TODO: Correct behavior of the dialer used in this method relies on the
// following pull request being merged: https://github.com/grpc/grpc-go/pull/974
// Once this pull-request is merged, we'll need to update our vendored versions
// of gRPC to enable this behavior. At the moment, the generated client will
// still try to dial indefinitely and will always fail.
func NewNonRedialingClientConnection(connection net.Conn) *grpc.ClientConn {
	// Create a one-shot dialer to use in client creation. This dialer will
	// return an error if invoked more than once, and gRPC will recognize that
	// error as non-temporary, thereby aborting any redials.
	connections := make(chan net.Conn, 1)
	connections <- connection
	close(connections)
	dialer := func(_ string, _ time.Duration) (net.Conn, error) {
		if c, ok := <-connections; ok {
			return c, nil
		}
		return nil, &oneShotDialerError{}
	}

	// Perform a dial, enforcing that this work the first time through, which it
	// always should. This enforcement makes the API a bit simpler.
	client, err := grpc.Dial("", grpc.WithBlock(), grpc.WithDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic(errors.Wrap(err, "in-memory dial failed"))
	}

	// Success.
	return client
}
