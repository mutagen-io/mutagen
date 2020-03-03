package local

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
)

// DisableLazyListenerInitialization indicates that lazy listener initialization
// should be disabled for all endpoints in the current process. It must be set
// during an init function and must not be changed later. This should only be
// used for integration tests, where lazy listener initialization needs to be
// disabled for proper coordination.
var DisableLazyListenerInitialization bool

// listenerEndpoint implements forwarding.Endpoint for listener endpoints. It
// optionally support lazy initialization.
type listenerEndpoint struct {
	// version is the forwarding session version.
	version forwarding.Version
	// configuration is the forwarding session configuration.
	configuration *forwarding.Configuration
	// protocol is the listening protocol
	protocol string
	// address is the listening address.
	address string
	// lazy indicates whether or not the endpoint uses lazy initialization.
	lazy bool
	// initializeOnce is used to guard calls to initialize.
	initializeOnce sync.Once
	// listener is the underlying listener. It is set by initialize.
	listener net.Listener
	// initializeError is any error that occurred during initialization.
	initializeError error
}

// NewListenerEndpoint creates a new forwarding.Endpoint that behaves as a
// listener. If lazy is true, then the underlying listener won't be initialized
// until the first call to Open. This is recommended for local endpoints since
// their primary use case is forwarding to remote endpoints whose connections
// may take longer to establish, thus leaving a window open where the accept
// backlog for the listener makes it appear that there's an active listener even
// though the forwarding loop isn't yet accepting connections. It is not
// recommended for remote endpoints because there's no good mechanism to report
// lazy initialization errors. Fortunately, for remote listeners, the most
// common use case is remote-to-local forwarding, meaning that there won't be
// much delay between the time the listener is established and the time that it
// starts accepting connections.
//
// TODO: We might want to create a better post-initialization error reporting
// mechanism for remote endpoints so that they can switch to using lazy
// initialization. This is pretty complicated since yamux owns the wire at that
// point and doesn't provide a mechanism for performing that error transmission.
// In any event, this would require a breaking change to the Mutagen forwarding
// protocol, and the benefit would be for a fairly niche scenario of a fairly
// niche use case (i.e. remote-to-remote forwarding where only the source
// endpoint connects and there is some application behaving badly while blocked
// on that listener's backlog). I think we should wait until there's some demand
// for that, but certainly consider proactively addressing it if we decide to
// refactor the forwarding protocol. In any case, this TODO belongs to the
// remote endpoint package rather than the local one.
func NewListenerEndpoint(
	version forwarding.Version,
	configuration *forwarding.Configuration,
	protocol string,
	address string,
	lazy bool,
) (forwarding.Endpoint, error) {
	// If lazy listener initialization has been globally disabled, then override
	// the requested mode.
	if DisableLazyListenerInitialization {
		lazy = false
	}

	// Create the endpoint.
	endpoint := &listenerEndpoint{
		version:       version,
		configuration: configuration,
		protocol:      protocol,
		address:       address,
		lazy:          lazy,
	}

	// Perform initialization if required.
	if !lazy {
		endpoint.initializeOnce.Do(func() { endpoint.initialize(false) })
		if endpoint.initializeError != nil {
			return nil, endpoint.initializeError
		}
	}

	// Done.
	return endpoint, nil
}

// initialize performs initialization for the endpoint. It will set either the
// listener member or listenError member. It should be invoked using the
// initializeOnce member.
func (e *listenerEndpoint) initialize(shutdown bool) {
	// If we're called on shutdown, then we act as a no-op.
	if shutdown {
		e.initializeError = errors.New("endpoint shutdown")
		return
	}

	// If we're dealing with a Windows named pipe target, then perform listening
	// using the platform-specific listening function.
	if e.protocol == "npipe" {
		e.listener, e.initializeError = listenWindowsNamedPipe(e.address)
		return
	}

	// Otherwise attempt to create a listener using the generic method.
	listener, err := net.Listen(e.protocol, e.address)
	if err != nil {
		// If we're not targeting a Unix domain socket or the error isn't due to
		// a conflicting socket, then abort.
		if e.protocol != "unix" || !isConflictingSocket(err) {
			e.initializeError = err
			return
		}

		// Compute the effective socket overwrite mode.
		socketOverwriteMode := e.configuration.SocketOverwriteMode
		if socketOverwriteMode.IsDefault() {
			socketOverwriteMode = e.version.DefaultSocketOverwriteMode()
		}

		// Check if a socket overwrite has been requested. If not, then abort.
		if !socketOverwriteMode.AttemptOverwrite() {
			e.initializeError = err
			return
		}

		// Attempt to remove the conflicting socket.
		if err := os.Remove(e.address); err != nil {
			e.initializeError = fmt.Errorf("unable to remove conflicting socket: %w", err)
			return
		}

		// Retry listening.
		listener, err = net.Listen(e.protocol, e.address)
		if err != nil {
			e.initializeError = fmt.Errorf("unable to create listener after conflicting socket removal: %w", err)
			return
		}
	}

	// If we're dealing with a Unix domain socket, then set ownership and
	// permissions.
	if e.protocol == "unix" {
		// Compute the effective socket owner specification.
		socketOwnerSpecification := e.configuration.SocketOwner
		if socketOwnerSpecification == "" {
			socketOwnerSpecification = e.version.DefaultSocketOwnerSpecification()
		}

		// Compute the effective socket group specification.
		socketGroupSpecification := e.configuration.SocketGroup
		if socketGroupSpecification == "" {
			socketGroupSpecification = e.version.DefaultSocketGroupSpecification()
		}

		// Compute the effective ownership specification.
		socketOwnership, err := filesystem.NewOwnershipSpecification(
			socketOwnerSpecification,
			socketGroupSpecification,
		)
		if err != nil {
			listener.Close()
			e.initializeError = fmt.Errorf("unable to create socket ownership specification: %w", err)
			return
		}

		// Compute the effective socket permission mode.
		socketPermissionMode := filesystem.Mode(e.configuration.SocketPermissionMode)
		if socketPermissionMode == 0 {
			socketPermissionMode = e.version.DefaultSocketPermissionMode()
		}

		// Set ownership and permissions.
		if err := filesystem.SetPermissionsByPath(e.address, socketOwnership, socketPermissionMode); err != nil {
			listener.Close()
			e.initializeError = fmt.Errorf("unable to set socket permissions: %w", err)
			return
		}
	}

	// Success.
	e.listener = listener
}

// TransportErrors implements forwarding.Endpoint.TransportErrors.
func (e *listenerEndpoint) TransportErrors() <-chan error {
	// Local endpoints don't have a transport that can fail, so we can return an
	// unbuffered empty channel that will never be populated.
	return make(chan error)
}

// Open implements forwarding.Endpoint.Open.
func (e *listenerEndpoint) Open() (net.Conn, error) {
	// For lazily initialized endpoints, we need to ensure that the listener has
	// been established.
	if e.lazy {
		e.initializeOnce.Do(func() { e.initialize(false) })
		if e.initializeError != nil {
			return nil, fmt.Errorf("lazy listen error: %w", e.initializeError)
		}
	}

	// Accept a connection.
	return e.listener.Accept()
}

// Shutdown implements forwarding.Endpoint.Shutdown.
func (e *listenerEndpoint) Shutdown() error {
	// For lazily initialized endpoints, it's possible that initialization
	// hasn't occurred yet. In these cases, attempt a "shutdown" initialization
	// to prevent any future initialization. If we succeed, or lazy
	// initialization had previously failed, then the listener will be nil and
	// there's nothing else we need to do.
	if e.lazy {
		e.initializeOnce.Do(func() { e.initialize(true) })
		if e.listener == nil {
			return nil
		}
	}

	// In all other cases (including those where lazy initialization has
	// succeeded) we know that a listener has been established, so we need to
	// close it.
	return e.listener.Close()
}
