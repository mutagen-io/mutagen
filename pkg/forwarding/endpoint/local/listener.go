package local

import (
	"fmt"
	"net"
	"os"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
)

// listenerEndpoint implements forwarding.Endpoint for listener endpoints.
type listenerEndpoint struct {
	// listener is the underlying listener.
	listener net.Listener
}

// NewListenerEndpoint creates a new forwarding.Endpoint that behaves as a
// listener.
func NewListenerEndpoint(
	version forwarding.Version,
	configuration *forwarding.Configuration,
	protocol string,
	address string,
) (forwarding.Endpoint, error) {
	// If we're dealing with a Windows named pipe target, then perform listening
	// using the platform-specific listening function.
	if protocol == "npipe" {
		listener, err := listenWindowsNamedPipe(address)
		if err != nil {
			return nil, err
		}
		return &listenerEndpoint{listener: listener}, nil
	}

	// Otherwise attempt to create a listener using the generic method.
	listener, err := net.Listen(protocol, address)
	if err != nil {
		// If we're not targeting a Unix domain socket or the error isn't due to
		// a conflicting socket, then abort.
		if protocol != "unix" || !isConflictingSocket(err) {
			return nil, err
		}

		// Compute the effective socket overwrite mode.
		socketOverwriteMode := configuration.SocketOverwriteMode
		if socketOverwriteMode.IsDefault() {
			socketOverwriteMode = version.DefaultSocketOverwriteMode()
		}

		// Check if a socket overwrite has been requested. If not, then abort.
		if !socketOverwriteMode.AttemptOverwrite() {
			return nil, err
		}

		// Attempt to remove the conflicting socket.
		if err := os.Remove(address); err != nil {
			return nil, fmt.Errorf("unable to remove existing socket: %w", err)
		}

		// Retry listening.
		listener, err = net.Listen(protocol, address)
		if err != nil {
			return nil, fmt.Errorf("unable to create listener after conflicting socket removal: %w", err)
		}
	}

	// If we're dealing with a Unix domain socket, then set ownership and
	// permissions.
	if protocol == "unix" {
		// Compute the effective socket owner specification.
		socketOwnerSpecification := configuration.SocketOwner
		if socketOwnerSpecification == "" {
			socketOwnerSpecification = version.DefaultSocketOwnerSpecification()
		}

		// Compute the effective socket group specification.
		socketGroupSpecification := configuration.SocketGroup
		if socketGroupSpecification == "" {
			socketGroupSpecification = version.DefaultSocketGroupSpecification()
		}

		// Compute the effective ownership specification.
		socketOwnership, err := filesystem.NewOwnershipSpecification(
			socketOwnerSpecification,
			socketGroupSpecification,
		)
		if err != nil {
			listener.Close()
			return nil, fmt.Errorf("unable to create socket ownership specification: %w", err)
		}

		// Compute the effective socket permission mode.
		socketPermissionMode := filesystem.Mode(configuration.SocketPermissionMode)
		if socketPermissionMode == 0 {
			socketPermissionMode = version.DefaultSocketPermissionMode()
		}

		// Set ownership and permissions.
		if err := filesystem.SetPermissionsByPath(address, socketOwnership, socketPermissionMode); err != nil {
			listener.Close()
			return nil, fmt.Errorf("unable to set socket permissions: %w", err)
		}
	}

	// Create the endpoint.
	return &listenerEndpoint{
		listener: listener,
	}, nil
}

// TransportErrors implements forwarding.Endpoint.TransportErrors.
func (e *listenerEndpoint) TransportErrors() <-chan error {
	// Local endpoints don't have a transport that can fail, so we can return an
	// unbuffered empty channel that will never be populated.
	return make(chan error)
}

// Open implements forwarding.Endpoint.Open.
func (e *listenerEndpoint) Open() (net.Conn, error) {
	return e.listener.Accept()
}

// Shutdown implements forwarding.Endpoint.Shutdown.
func (e *listenerEndpoint) Shutdown() error {
	return e.listener.Close()
}
