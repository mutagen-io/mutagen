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

type Server struct {
	// Termination is populated with requests from clients invoking the shutdown
	// method over RPC. It can be ignored by daemon host processes wishing to
	// ignore temination requests originating from clients. The channel is
	// buffered and non-blocking, so it doesn't need to be serviced by the
	// daemon host-process at all - additional incoming shutdown requests will
	// just bounce off once the channel is populated. We do this, instead of
	// closing the channel, because we can't close the channel multiple times.
	Termination chan struct{}
	// housekeepingTicker is a ticker that regulates housekeeping.
	housekeepingTicker *time.Ticker
	// housekeepingCancel is a context cancellation function used to stop the
	// housekeeping Goroutine. This is necessary because tickers don't close
	// their internal channels when stopped.
	housekeepingCancel context.CancelFunc
}

func New() *Server {
	// Create the housekeeping ticker.
	housekeepingTicker := time.NewTicker(housekeepingInterval)

	// Create the housekeeping context.
	housekeepingContext, housekeepingCancel := context.WithCancel(context.Background())

	// Start housekeeping in a separate Goroutine.
	go func() {
		for {
			select {
			case <-housekeepingContext.Done():
				return
			case <-housekeepingTicker.C:
				agent.Housekeep()
				session.HousekeepCaches()
				session.HousekeepStaging()
			}
		}
	}()

	// Create the server.
	return &Server{
		Termination:        make(chan struct{}, 1),
		housekeepingTicker: housekeepingTicker,
		housekeepingCancel: housekeepingCancel,
	}
}

func (s *Server) Shutdown() {
	// Stop the housekeeping ticker.
	s.housekeepingTicker.Stop()

	// Cancel housekeeping.
	s.housekeepingCancel()
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
