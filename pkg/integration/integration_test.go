package integration

import (
	"io"
	"log"
	"os"
	"testing"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/daemon"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/ipc"
	"github.com/mutagen-io/mutagen/pkg/logging"
	daemonsvc "github.com/mutagen-io/mutagen/pkg/service/daemon"
	forwardingsvc "github.com/mutagen-io/mutagen/pkg/service/forwarding"
	promptingsvc "github.com/mutagen-io/mutagen/pkg/service/prompting"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"

	// Explicitly import packages that need to register protocol handlers.
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/docker"
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/local"
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/ssh"
	_ "github.com/mutagen-io/mutagen/pkg/integration/protocols/netpipe"
	_ "github.com/mutagen-io/mutagen/pkg/synchronization/protocols/docker"
	_ "github.com/mutagen-io/mutagen/pkg/synchronization/protocols/local"
	_ "github.com/mutagen-io/mutagen/pkg/synchronization/protocols/ssh"
)

// forwardingManager is the forwarding session manager for the integration
// testing daemon. It is exposed for integration tests that operate at the API
// level (as opposed to the gRPC or command line level).
var forwardingManager *forwarding.Manager

// synchronizationManager is the synchronization session manager for the
// integration testing daemon. It is exposed for integration tests that operate
// at the API level (as opposed to the gRPC or command line level).
var synchronizationManager *synchronization.Manager

// TestMain is the entry point for integration tests. It replaces the default
// test entry point so that it can copy the mutagen executable to a well-known
// path, set up the agent bundle to work during testing, set up a functionally
// complete daemon instance for testing, and tear down all of the aforementioned
// infrastructure after running tests.
func TestMain(m *testing.M) {
	// Disable logging.
	log.SetOutput(io.Discard)

	// Override the expected agent bundle location.
	agent.ExpectedBundleLocation = agent.BundleLocationBuildDirectory

	// Acquire the daemon lock and defer its release.
	lock, err := daemon.AcquireLock()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to acquire daemon lock"))
	}
	defer lock.Release()

	// Create a forwarding session manager and defer its shutdown.
	forwardingManager, err = forwarding.NewManager(logging.RootLogger.Sublogger("forwarding"))
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create forwarding session manager"))
	}
	defer forwardingManager.Shutdown()

	// Create a session manager and defer its shutdown.
	synchronizationManager, err = synchronization.NewManager(logging.RootLogger.Sublogger("sync"))
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create synchronization session manager"))
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
	promptingsvc.RegisterPromptingServer(server, promptingsvc.NewServer())

	// Create and register the forwarding server.
	forwardingServer := forwardingsvc.NewServer(forwardingManager)
	forwardingsvc.RegisterForwardingServer(server, forwardingServer)

	// Create and register the session service.
	synchronizationServer := synchronizationsvc.NewServer(synchronizationManager)
	synchronizationsvc.RegisterSynchronizationServer(server, synchronizationServer)

	// Compute the path to the daemon IPC endpoint.
	endpoint, err := daemon.EndpointPath()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to compute endpoint path"))
	}

	// Create the daemon listener and defer its closure. Since we hold the
	// daemon lock, we preemptively remove any existing socket since it (should)
	// be stale.
	os.Remove(endpoint)
	listener, err := ipc.NewListener(endpoint)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create daemon listener"))
	}
	defer listener.Close()

	// Serve incoming connections in a separate Goroutine. We don't monitor for
	// errors since there's nothing that we can do about them and because
	// they'll likely show up in the test output anyway.
	go server.Serve(listener)

	// Run tests.
	m.Run()
}
