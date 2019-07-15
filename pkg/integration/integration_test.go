package integration

import (
	"os"
	"testing"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/daemon"
	"github.com/havoc-io/mutagen/pkg/forwarding"
	"github.com/havoc-io/mutagen/pkg/grpcutil"
	"github.com/havoc-io/mutagen/pkg/ipc"
	"github.com/havoc-io/mutagen/pkg/logging"
	daemonsvc "github.com/havoc-io/mutagen/pkg/service/daemon"
	forwardingsvc "github.com/havoc-io/mutagen/pkg/service/forwarding"
	promptsvc "github.com/havoc-io/mutagen/pkg/service/prompt"
	synchronizationsvc "github.com/havoc-io/mutagen/pkg/service/synchronization"
	"github.com/havoc-io/mutagen/pkg/synchronization"

	// Explicitly import packages that need to register protocol handlers.
	_ "github.com/havoc-io/mutagen/pkg/forwarding/protocols/docker"
	_ "github.com/havoc-io/mutagen/pkg/forwarding/protocols/local"
	_ "github.com/havoc-io/mutagen/pkg/forwarding/protocols/ssh"
	_ "github.com/havoc-io/mutagen/pkg/integration/protocols/netpipe"
	_ "github.com/havoc-io/mutagen/pkg/synchronization/protocols/docker"
	_ "github.com/havoc-io/mutagen/pkg/synchronization/protocols/local"
	_ "github.com/havoc-io/mutagen/pkg/synchronization/protocols/ssh"
)

// forwardingManager is the forwarding session manager for the integration
// testing daemon. It is exposed for integration tests that operate at the API
// level (as opposed to the gRPC or command line level).
var forwardingManager *forwarding.Manager

// synchronizationManager is the synchronization session manager for the
// integration testing daemon. It is exposed for integration tests that operate
// at the API level (as opposed to the gRPC or command line level).
var synchronizationManager *synchronization.Manager

// testMainInternal is the internal testing entry point, needed so that shutdown
// operations can be deferred (since TestMain will invoke os.Exit). It copies
// the mutagen executable to a well-known path, sets up the agent bundle to work
// during testing, sets up a functionally complete daemon instance for testing,
// runs integration tests, and finally tears down all of the aforementioned
// infrastructure.
func testMainInternal(m *testing.M) (int, error) {
	// Override the expected agent bundle location.
	agent.ExpectedBundleLocation = agent.BundleLocationBuildDirectory

	// Acquire the daemon lock and defer its release.
	lock, err := daemon.AcquireLock()
	if err != nil {
		return -1, errors.Wrap(err, "unable to acquire daemon lock")
	}
	defer lock.Release()

	// Create a forwarding session manager and defer its shutdown.
	forwardingManager, err = forwarding.NewManager(logging.RootLogger.Sublogger("forwarding"))
	if err != nil {
		return -1, errors.Wrap(err, "unable to create forwarding session manager")
	}
	defer forwardingManager.Shutdown()

	// Create a session manager and defer its shutdown. Note that we assign to
	// the global instance here.
	synchronizationManager, err = synchronization.NewManager(logging.RootLogger.Sublogger("sync"))
	if err != nil {
		return -1, errors.Wrap(err, "unable to create synchronization session manager")
	}
	defer synchronizationManager.Shutdown()

	// Create the gRPC server and defer its stoppage. We use a hard stop rather
	// than a graceful stop so that it doesn't hang on open requests.
	server := grpc.NewServer(
		grpc.MaxSendMsgSize(grpcutil.MaximumMessageSize),
		grpc.MaxRecvMsgSize(grpcutil.MaximumMessageSize),
	)
	defer server.Stop()

	// Create and register the daemon service and defer its shutdown.
	daemonServer := daemonsvc.NewServer()
	daemonsvc.RegisterDaemonServer(server, daemonServer)
	defer daemonServer.Shutdown()

	// Create and register the prompt service.
	promptsvc.RegisterPromptingServer(server, promptsvc.NewServer())

	// Create and register the forwarding server.
	forwardingServer := forwardingsvc.NewServer(forwardingManager)
	forwardingsvc.RegisterForwardingServer(server, forwardingServer)

	// Create and register the session service.
	synchronizationServer := synchronizationsvc.NewServer(synchronizationManager)
	synchronizationsvc.RegisterSynchronizationServer(server, synchronizationServer)

	// Compute the path to the daemon IPC endpoint.
	endpoint, err := daemon.EndpointPath()
	if err != nil {
		return -1, errors.Wrap(err, "unable to compute endpoint path")
	}

	// Create the daemon listener and defer its closure. Since we hold the
	// daemon lock, we preemptively remove any existing socket since it (should)
	// be stale.
	os.Remove(endpoint)
	listener, err := ipc.NewListener(endpoint)
	if err != nil {
		return -1, errors.Wrap(err, "unable to create daemon listener")
	}
	defer listener.Close()

	// Serve incoming connections in a separate Goroutine. We don't monitor for
	// errors since there's nothing that we can do about them and because
	// they'll likely show up in the test output anyway.
	go server.Serve(listener)

	// Run tests.
	return m.Run(), nil
}

// TestMain is the entry point for integration tests (overriding the default
// generated entry point).
func TestMain(m *testing.M) {
	// Invoke the internal entry point. If there's an error, print it out before
	// exiting.
	result, err := testMainInternal(m)
	if err != nil {
		cmd.Error(err)
	}

	// Exit with the result.
	os.Exit(result)
}
