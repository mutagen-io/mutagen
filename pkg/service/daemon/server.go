package daemon

import (
	"context"
	"time"

	"github.com/mutagen-io/mutagen/pkg/housekeeping"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
)

const (
	// housekeepingInterval is the interval at which housekeeping will be
	// invoked by the daemon.
	housekeepingInterval = 24 * time.Hour
)

// Server provides an implementation of the Daemon service.
type Server struct {
	// UnimplementedDaemonServer is the required base implementation.
	UnimplementedDaemonServer
	// Termination is populated with requests from clients invoking the shutdown
	// method over RPC. It can be ignored by daemon host processes wishing to
	// ignore termination requests originating from clients. The channel is
	// buffered and non-blocking, so it doesn't need to be serviced by the
	// daemon host-process at all - additional incoming shutdown requests will
	// just bounce off once the channel is populated. We do this, instead of
	// closing the channel, because we can't close the channel multiple times.
	Termination chan struct{}
	// workerCtx is the context regulating the server's internal operations.
	workerCtx context.Context
	// shutdown is the context cancellation function for the server's internal
	// operation context.
	shutdown context.CancelFunc

	logger *logging.Logger
}

// NewServer creates a new daemon server.
func NewServer(logger *logging.Logger) *Server {
	// Create a cancellable context for daemon background operations.
	workerCtx, shutdown := context.WithCancel(context.Background())

	// Create the server.
	server := &Server{
		Termination: make(chan struct{}, 1),
		workerCtx:   workerCtx,
		shutdown:    shutdown,
		logger:      logger,
	}

	// Start the housekeeping Goroutine.
	go server.housekeep()

	// Done.
	return server
}

// housekeep provides regular housekeeping facilities for the daemon.
func (s *Server) housekeep() {
	// Perform an initial housekeeping operation since the ticker won't fire
	// straight away.
	housekeeping.Housekeep(s.logger)

	// Create a ticker to regulate housekeeping and defer its shutdown.
	ticker := time.NewTicker(housekeepingInterval)
	defer ticker.Stop()

	// Loop and wait for the ticker or cancellation.
	for {
		select {
		case <-s.workerCtx.Done():
			return
		case <-ticker.C:
			housekeeping.Housekeep(s.logger)
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
	return &VersionResponse{
		Major: mutagen.VersionMajor,
		Minor: mutagen.VersionMinor,
		Patch: mutagen.VersionPatch,
		Tag:   mutagen.VersionTag,
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
