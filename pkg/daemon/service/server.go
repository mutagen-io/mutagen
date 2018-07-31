package daemon

import (
	"context"
	"sync"
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

// Server provides an implementation of the Daemon service, providing methods
// for managing the daemon lifecycle. This Server is designed to operate as a
// singleton and can be accessed via the global DefaultServer variable.
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

// defaultServerLock controls access to the defaultServer variable.
var defaultServerLock sync.RWMutex

// defaultServer is the default daemon server.
var defaultServer *Server

// DefaultServer provides the default daemon server, creating it if necessary.
func DefaultServer() *Server {
	// Optimistically attempt to grab the server.
	defaultServerLock.RLock()
	if defaultServer != nil {
		defer defaultServerLock.RUnlock()
		return defaultServer
	}
	defaultServerLock.RUnlock()

	// Otherwise we need to create the server, so we'll need to get a write
	// lock on the server.
	defaultServerLock.Lock()
	defer defaultServerLock.Unlock()

	// It's possible that the server was created by someone else between our two
	// lockings, so see if we can just return it.
	if defaultServer != nil {
		return defaultServer
	}

	// Create the default daemon context.
	context, shutdown := context.WithCancel(context.Background())

	// Create the default daemon server.
	defaultServer = &Server{
		Termination: make(chan struct{}, 1),
		context:     context,
		shutdown:    shutdown,
	}

	// Start the housekeeping Goroutine.
	go defaultServer.housekeep()

	// Done.
	return defaultServer
}

// housekeep provides regular housekeeping facilities for the daemon.
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

// Shutdown gracefully shuts down server resources.
func (s *Server) Shutdown() {
	// Cancel all internal operations.
	s.shutdown()
}

// Version provides version information.
func (s *Server) Version(_ context.Context, _ *VersionRequest) (*VersionResponse, error) {
	// Send the version response.
	return &VersionResponse{
		Major: mutagen.VersionMajor,
		Minor: mutagen.VersionMinor,
		Patch: mutagen.VersionPatch,
	}, nil
}

// Terminate requests daemon termination.
func (s *Server) Terminate(_ context.Context, _ *TerminateRequest) (*TerminateResponse, error) {
	// Send the termination request in a non-blocking manner.
	select {
	case s.Termination <- struct{}{}:
	default:
	}

	// Success.
	return &TerminateResponse{}, nil
}
