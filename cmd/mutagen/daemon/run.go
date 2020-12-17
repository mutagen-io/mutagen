package daemon

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"

	"github.com/mutagen-io/mutagen/cmd"

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

	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/docker"
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/local"
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/ssh"
	_ "github.com/mutagen-io/mutagen/pkg/synchronization/protocols/docker"
	_ "github.com/mutagen-io/mutagen/pkg/synchronization/protocols/local"
	_ "github.com/mutagen-io/mutagen/pkg/synchronization/protocols/ssh"
)

// runMain is the entry point for the run command.
func runMain(_ *cobra.Command, _ []string) error {
	// Attempt to acquire the daemon lock and defer its release.
	lock, err := daemon.AcquireLock()
	if err != nil {
		return errors.Wrap(err, "unable to acquire daemon lock")
	}
	defer lock.Release()

	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Create a forwarding session manager and defer its shutdown.
	forwardingManager, err := forwarding.NewManager(logging.RootLogger.Sublogger("forwarding"))
	if err != nil {
		return errors.Wrap(err, "unable to create forwarding session manager")
	}
	defer forwardingManager.Shutdown()

	// Create a synchronization session manager and defer its shutdown.
	synchronizationManager, err := synchronization.NewManager(logging.RootLogger.Sublogger("synchronization"))
	if err != nil {
		return errors.Wrap(err, "unable to create synchronization session manager")
	}
	defer synchronizationManager.Shutdown()

	// Create the gRPC server and defer its stoppage. We use a hard stop rather
	// than a graceful stop so that it doesn't hang on open requests.
	server := grpc.NewServer(
		grpc.MaxSendMsgSize(grpcutil.MaximumMessageSize),
		grpc.MaxRecvMsgSize(grpcutil.MaximumMessageSize),
	)
	defer server.Stop()

	// Create the daemon server, defer its shutdown, and register it.
	daemonServer := daemonsvc.NewServer()
	defer daemonServer.Shutdown()
	daemonsvc.RegisterDaemonServer(server, daemonServer)

	// Create and register the prompt server.
	promptingsvc.RegisterPromptingServer(server, promptingsvc.NewServer())

	// Create and register the forwarding server.
	forwardingServer := forwardingsvc.NewServer(forwardingManager)
	forwardingsvc.RegisterForwardingServer(server, forwardingServer)

	// Create and register the synchronization server.
	synchronizationServer := synchronizationsvc.NewServer(synchronizationManager)
	synchronizationsvc.RegisterSynchronizationServer(server, synchronizationServer)

	// Compute the path to the daemon IPC endpoint.
	endpoint, err := daemon.EndpointPath()
	if err != nil {
		return errors.Wrap(err, "unable to compute endpoint path")
	}

	// Create the daemon listener and defer its closure. Since we hold the
	// daemon lock, we preemptively remove any existing socket since it (should)
	// be stale.
	os.Remove(endpoint)
	listener, err := ipc.NewListener(endpoint)
	if err != nil {
		return errors.Wrap(err, "unable to create daemon listener")
	}
	defer listener.Close()

	// Serve incoming connections in a separate Goroutine, watching for serving
	// failure.
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(listener)
	}()

	// Wait for termination from a signal, the daemon service, or the gRPC
	// server. We treat termination via the daemon service as a non-error.
	select {
	case sig := <-signalTermination:
		return errors.Errorf("terminated by signal: %s", sig)
	case <-daemonServer.Termination:
		return nil
	case err = <-serverErrors:
		return errors.Wrap(err, "daemon server termination")
	}
}

// runCommand is the run command.
var runCommand = &cobra.Command{
	Use:          "run",
	Short:        "Run the Mutagen daemon",
	Args:         cmd.DisallowArguments,
	Hidden:       true,
	RunE:         runMain,
	SilenceUsage: true,
}

// runConfiguration stores configuration for the run command.
var runConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := runCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&runConfiguration.help, "help", "h", false, "Show help information")
}
