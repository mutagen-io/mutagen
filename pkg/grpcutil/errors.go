package grpcutil

import (
	"github.com/pkg/errors"

	"google.golang.org/grpc/status"
)

// PeelAwayRPCErrorLayer peels away any intermediate RPC error layer from an
// error returned by gRPC-based code and constructs an error object using the
// underlying error message. If this unwrapping fails, the argument is returned
// directly.
func PeelAwayRPCErrorLayer(err error) error {
	// Attempt to peel away the RPC layer.
	if s, ok := status.FromError(err); ok {
		return errors.New(s.Message())
	}

	// Otherwise return the argument directly.
	return err
}
