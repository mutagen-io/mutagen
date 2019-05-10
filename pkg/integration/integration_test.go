package integration

import (
	"os"
	"testing"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/daemon"
	daemonsvc "github.com/havoc-io/mutagen/pkg/service/daemon"
	promptsvc "github.com/havoc-io/mutagen/pkg/service/prompt"
	sessionsvc "github.com/havoc-io/mutagen/pkg/service/session"
	"github.com/havoc-io/mutagen/pkg/session"

	// Explicitly import packages that need to register protocol handlers.
	_ "github.com/havoc-io/mutagen/pkg/integration/protocols/netpipe"
	_ "github.com/havoc-io/mutagen/pkg/protocols/docker"
	_ "github.com/havoc-io/mutagen/pkg/protocols/local"
	_ "github.com/havoc-io/mutagen/pkg/protocols/ssh"
)

// sessionManager is the session manager for the integration testing daemon. It
// is exposed for integration tests that operate at the API level (as opposed to
// the gRPC or command line level).
var sessionManager *session.Manager

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
	defer lock.Unlock()

	// Create a session manager and defer its shutdown. Note that we assign to
	// the global instance here.
	sessionManager, err = session.NewManager()
	if err != nil {
		return -1, errors.Wrap(err, "unable to create session manager")
	}
	defer sessionManager.Shutdown()

	// Create the gRPC server and defer its stoppage. We use a hard stop rather
	// than a graceful stop so that it doesn't hang on open requests.
	server := grpc.NewServer(
		grpc.MaxSendMsgSize(daemon.MaximumIPCMessageSize),
		grpc.MaxRecvMsgSize(daemon.MaximumIPCMessageSize),
	)
	defer server.Stop()

	// Create and register the daemon service and defer its shutdown.
	daemonServer := daemonsvc.NewServer()
	daemonsvc.RegisterDaemonServer(server, daemonServer)
	defer daemonServer.Shutdown()

	// Create and register the prompt service.
	promptsvc.RegisterPromptingServer(server, promptsvc.NewServer())

	// Create and register the session service.
	sessionsvc.RegisterSessionsServer(server, sessionsvc.NewServer(sessionManager))

	// Create the daemon listener and defer its closure.
	listener, err := daemon.NewListener()
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
