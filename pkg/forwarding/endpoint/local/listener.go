package local

import (
	"net"
	"os"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/forwarding"
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
	// Compute the effective socket overwrite mode.
	socketOverwriteMode := configuration.SocketOverwriteMode
	if socketOverwriteMode.IsDefault() {
		socketOverwriteMode = version.DefaultSocketOverwriteMode()
	}

	// Create the underlying listener. If this is a Unix domain socket listener
	// and we fail due to an existing file, then attempt a removal and re-listen
	// if requested.
	listener, err := net.Listen(protocol, address)
	if err != nil {
		// HACK: os.IsExist doesn't seem to recognize the error here, so we
		// don't perform that check. This may be fixed in Go 1.13.
		attemptOverwrite := protocol == "unix" &&
			socketOverwriteMode == forwarding.SocketOverwriteMode_SocketOverwriteModeOverwrite
		if attemptOverwrite {
			if err := os.Remove(address); err != nil {
				return nil, errors.Wrap(err, "unable to overwrite existing socket")
			}
			listener, err = net.Listen(protocol, address)
			if err != nil {
				return nil, errors.Wrap(err, "unable to create listener after socket overwrite")
			}
		} else {
			return nil, errors.Wrap(err, "unable to create listener")
		}
	}

	// If we're dealing with a Unix domain socket, then handle its permissions
	// and ownership.
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
			return nil, errors.Wrap(err, "unable to create socket ownership specification")
		}

		// Compute the effective socket permission mode.
		socketPermissionMode := filesystem.Mode(configuration.SocketPermissionMode)
		if socketPermissionMode == 0 {
			socketPermissionMode = version.DefaultSocketPermissionMode()
		}

		// Set ownership and permissions.
		if err := filesystem.SetPermissionsByPath(address, socketOwnership, socketPermissionMode); err != nil {
			listener.Close()
			return nil, errors.Wrap(err, "unable to set socket permissions")
		}
	}

	// Create the endpoint.
	return &listenerEndpoint{
		listener: listener,
	}, nil
}

// Open implements forwarding.Endpoint.Open.
func (e *listenerEndpoint) Open() (net.Conn, error) {
	return e.listener.Accept()
}

// Shutdown implements forwarding.Endpoint.Shutdown.
func (e *listenerEndpoint) Shutdown() error {
	return e.listener.Close()
}
