package daemon

import (
	"github.com/havoc-io/mutagen/pkg/rpc"
)

const (
	MethodTerminate = "daemon.Terminate"
)

type Service struct {
	// termination is populated with requests from clients invoking the shutdown
	// method over RPC. It can be ignored by daemon host processes wishing to
	// ignore temination requests originating from clients. The channel is
	// buffered and non-blocking, so it doesn't need to be serviced by the
	// daemon host-process at all - additional incoming shutdown requests will
	// just bounce off once the channel is populated. We do this, instead of
	// closing the channel, because we can't close the channel multiple times.
	termination chan struct{}
}

func NewService() (*Service, chan struct{}) {
	// Create a termination channel.
	termination := make(chan struct{}, 1)

	// Create the service.
	return &Service{termination}, termination
}

func (s *Service) Methods() map[string]rpc.Handler {
	return map[string]rpc.Handler{
		MethodTerminate: s.Terminate,
	}
}

func (s *Service) Terminate(_ rpc.HandlerStream) error {
	// Send the termination request in a non-blocking manner.
	select {
	case s.termination <- struct{}{}:
	default:
	}

	// Done.
	return nil
}
