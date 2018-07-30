package service

import (
	"context"
	"time"

	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/session"
)

const (
	// housekeepingInterval is the interval at which housekeeping will be
	// invoked by the daemon.
	housekeepingInterval = 24 * time.Hour
)

// housekeep performs a combined housekeeping operation.
func housekeep() {
	// Perform agent housekeeping.
	agent.Housekeep()

	// Perform cache housekeeping.
	session.HousekeepCaches()

	// Perform staging directory housekeeping.
	session.HousekeepStaging()
}

type Server struct {
	// Termination is populated with requests from clients invoking the shutdown
	// method over RPC. It can be ignored by daemon host processes wishing to
	// ignore temination requests originating from clients. The channel is
	// buffered and non-blocking, so it doesn't need to be serviced by the
	// daemon host-process at all - additional incoming shutdown requests will
	// just bounce off once the channel is populated. We do this, instead of
	// closing the channel, because we can't close the channel multiple times.
	Termination chan struct{}
	// context is the context regulating the server's internal operations.
	context context.Context
	// shutdown is the context cancellation function for the server's internal
	// operation context.
	shutdown context.CancelFunc
}

func New() *Server {
	// Create the internal context.
	context, shutdown := context.WithCancel(context.Background())

	// Create the server.
	server := &Server{
		Termination: make(chan struct{}, 1),
		context:     context,
		shutdown:    shutdown,
	}

	// Start the housekeeping Goroutine.
	go server.housekeep()

	// Done.
	return server
}

func (s *Server) housekeep() {
	// Perform an initial housekeeping operation since the ticker won't fire
	// straight away.
	housekeep()

	// Create a ticker to regulate housekeeping and defer its shutdown.
	ticker := time.NewTicker(housekeepingInterval)
	defer ticker.Stop()

	// Loop and wait for the ticker or cancellation.
	for {
		select {
		case <-s.context.Done():
			return
		case <-ticker.C:
			housekeep()
		}
	}
}

func (s *Server) Shutdown() {
	// Cancel all internal operations.
	s.shutdown()
}

func (s *Server) Version(_ context.Context, _ *VersionRequest) (*VersionResponse, error) {
	// Send the version response.
	return &VersionResponse{
		Major: mutagen.VersionMajor,
		Minor: mutagen.VersionMinor,
		Patch: mutagen.VersionPatch,
	}, nil
}

func (s *Server) Terminate(_ context.Context, _ *TerminateRequest) (*TerminateResponse, error) {
	// Send the termination request in a non-blocking manner.
	select {
	case s.Termination <- struct{}{}:
	default:
	}

	// Success.
	return &TerminateResponse{}, nil
}
