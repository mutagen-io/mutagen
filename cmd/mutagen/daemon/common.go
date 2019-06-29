package daemon

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/pkg/daemon"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	daemonsvcpkg "github.com/havoc-io/mutagen/pkg/service/daemon"
)

const (
	// dialTimeout is the timeout to use when attempting to connect to the
	// daemon IPC endpoint.
	dialTimeout = 1 * time.Second
)

// daemonDialer is an adapter around daemon IPC dialing that fits gRPC's dialing
// interface. It ignores the provided address since the daemon package already
// knows the correct endpoint.
func daemonDialer(_ string, timeout time.Duration) (net.Conn, error) {
	return daemon.DialTimeout(timeout)
}

// CreateClientConnection creates a new daemon client connection and optionally
// verifies that the daemon version matches the current process' version.
func CreateClientConnection(enforceVersionMatch bool) (*grpc.ClientConn, error) {
	// Create a context to timeout the dial.
	dialContext, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	// Perform dialing.
	connection, err := grpc.DialContext(
		dialContext, "",
		grpc.WithInsecure(),
		grpc.WithDialer(daemonDialer),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(daemon.MaximumIPCMessageSize)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(daemon.MaximumIPCMessageSize)),
	)
	if err != nil {
		if err == context.DeadlineExceeded {
			return nil, errors.New("connection timed out (is the daemon running?)")
		}
		return nil, err
	}

	// If requested, verify that the daemon version matches the current process'
	// version. We'll perform this call within the dialing context since it
	// should be more than long enough to dial the daemon and perform a version
	// check.
	if enforceVersionMatch {
		daemonService := daemonsvcpkg.NewDaemonClient(connection)
		version, err := daemonService.Version(dialContext, &daemonsvcpkg.VersionRequest{})
		if err != nil {
			connection.Close()
			return nil, errors.Wrap(err, "unable to query daemon version")
		}
		versionMatch := version.Major == mutagen.VersionMajor &&
			version.Minor == mutagen.VersionMinor &&
			version.Patch == mutagen.VersionPatch &&
			version.Tag == mutagen.VersionTag
		if !versionMatch {
			connection.Close()
			return nil, errors.New("client/daemon version mismatch (daemon restart recommended)")
		}
	}

	// Success.
	return connection, nil
}
