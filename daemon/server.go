package daemon

import (
	"golang.org/x/net/context"
)

type Server struct {
	// Termination is populated with requests from clients invoking the shutdown
	// method over RPC. It can be ignored by daemon host processes wishing to
	// ignore temination requests originating from clients. The channel is
	// buffered and non-blocking, so it doesn't need to be serviced by the
	// daemon host-process at all - additional incoming shutdown requests will
	// just bounce off once the channel is populated.
	Termination chan struct{}
}

func NewServer() *Server {
	return &Server{
		Termination: make(chan struct{}, 1),
	}
}

func (s *Server) Shutdown(_ context.Context, _ *ShutdownRequest) (*ShutdownResponse, error) {
	// Send the termination request in a non-blocking manner. If there is
	// already a termination request in the pipeline, this method is a no-op.
	select {
	case s.Termination <- struct{}{}:
	default:
	}

	// Done.
	return &ShutdownResponse{}, nil
}
