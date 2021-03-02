package netpipe

import (
	"context"
	"net"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization/endpoint/remote"
	urlpkg "github.com/mutagen-io/mutagen/pkg/url"
)

// synchronizationProtocolHandler implements the synchronization.ProtocolHandler
// interface for connecting to "remote" endpoints that actually exist in memory
// via an in-memory pipe.
type synchronizationProtocolHandler struct{}

// waitingSynchronizationEndpoint wraps and implements synchronization.Endpoint,
// but adds a waiting function that's invoked after invoking Shutdown on the
// underlying endpoint. It is necessary to ensure full endpoint shutdown in
// tests, where open file descriptors or handles can prevent temporary directory
// removal.
type waitingSynchronizationEndpoint struct {
	// Endpoint is the underlying endpoint.
	synchronization.Endpoint
	// wait is an arbitrary waiting function.
	wait func()
}

// Shutdown implements synchronization.Endpoint.Shutdown.
func (w *waitingSynchronizationEndpoint) Shutdown() error {
	result := w.Endpoint.Shutdown()
	w.wait()
	return result
}

// Dial starts an endpoint server in a background Goroutine and creates an
// endpoint client connected to the server via an in-memory connection.
func (h *synchronizationProtocolHandler) Connect(
	_ context.Context,
	logger *logging.Logger,
	url *urlpkg.URL,
	prompter string,
	session string,
	version synchronization.Version,
	configuration *synchronization.Configuration,
	alpha bool,
) (synchronization.Endpoint, error) {
	// Verify that the URL is of the correct kind and protocol.
	if url.Kind != urlpkg.Kind_Synchronization {
		panic("non-synchronization URL dispatched to synchronization protocol handler")
	} else if url.Protocol != Protocol_Netpipe {
		panic("non-netpipe URL dispatched to netpipe protocol handler")
	}

	// Create an in-memory network connection.
	clientConnection, serverConnection := net.Pipe()

	// Serve the endpoint in a background Goroutine. This will terminate once
	// the client connection is closed. We monitor for its termination so that
	// we can block on it in our endpoint wrapper.
	remoteEndpointDone := make(chan struct{})
	go func() {
		remote.ServeEndpoint(logger.Sublogger("remote"), serverConnection)
		close(remoteEndpointDone)
	}()

	// Create a client for this endpoint.
	endpoint, err := remote.NewEndpoint(
		clientConnection,
		url.Path,
		session,
		version,
		configuration,
		alpha,
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create in-memory endpoint client")
	}

	// Wrap the client so that it blocks on the full shutdown of the remote
	// endpoint after closing the connection. This is necessary for testing,
	// where we need to ensure that all file descriptors or handles point to
	// temporary test directories are closed before attempting to remove those
	// directories. This is not necessary for other remote protocols in normal
	// usage (because we don't have the same constraints) or in testing (because
	// the underlying connection closure waits for agent process termination).
	endpoint = &waitingSynchronizationEndpoint{endpoint, func() { <-remoteEndpointDone }}

	// Success.
	return endpoint, nil
}

func init() {
	// Register the netpipe protocol handler with the synchronization package.
	synchronization.ProtocolHandlers[Protocol_Netpipe] = &synchronizationProtocolHandler{}
}
