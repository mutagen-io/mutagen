package daemon

import (
	"golang.org/x/net/context"
)

type service struct {
	// termination is populated with requests from clients invoking the shutdown
	// method over RPC. It can be ignored by daemon host processes wishing to
	// ignore temination requests originating from clients. The channel is
	// buffered and non-blocking, so it doesn't need to be serviced by the
	// daemon host-process at all - additional incoming shutdown requests will
	// just bounce off once the channel is populated.
	termination chan struct{}
}

func newService() (*service, chan struct{}) {
	// Create a termination channel.
	termination := make(chan struct{}, 1)

	// Create the service.
	return &service{termination}, termination
}

func (s *service) Terminate(_ context.Context, _ *TerminateRequest) (*TerminateResponse, error) {
	// Send the termination request in a non-blocking manner. If there is
	// already a termination request in the pipeline, this method is a no-op.
	select {
	case s.termination <- struct{}{}:
	default:
	}

	// Done.
	return &TerminateResponse{}, nil
}
