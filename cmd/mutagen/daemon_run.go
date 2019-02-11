package main

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
	mgrpc "github.com/havoc-io/mutagen/pkg/grpc"
	daemonsvc "github.com/havoc-io/mutagen/pkg/service/daemon"
	promptsvc "github.com/havoc-io/mutagen/pkg/service/prompt"
	sessionsvc "github.com/havoc-io/mutagen/pkg/service/session"
)

func daemonRunMain(command *cobra.Command, arguments []string) error {
	// Validate arguments.
	if len(arguments) != 0 {
		return errors.New("unexpected arguments provided")
	}

	// Attempt to acquire the daemon lock and defer its release. If there is a
	// crash, the lock will be released by the OS automatically, but on Windows
	// this may only happen after some unspecified period of time (though it
	// does seem to be basically instant).
	lock, err := daemon.AcquireLock()
	if err != nil {
		return errors.Wrap(err, "unable to acquire daemon lock")
	}
	defer lock.Unlock()

	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Create the gRPC server.
	server := grpc.NewServer(
		grpc.MaxSendMsgSize(mgrpc.MaximumIPCMessageSize),
		grpc.MaxRecvMsgSize(mgrpc.MaximumIPCMessageSize),
	)

	// Create and register the daemon service and defer its shutdown.
	daemonServer := daemonsvc.New()
	daemonsvc.RegisterDaemonServer(server, daemonServer)
	defer daemonServer.Shutdown()

	// Create and register the prompt service.
	promptsvc.RegisterPromptingServer(server, promptsvc.New())

	// Create and register the session service and defer its shutdown.
	sessionsServer, err := sessionsvc.New()
	if err != nil {
		return errors.Wrap(err, "unable to create sessions service")
	}
	sessionsvc.RegisterSessionsServer(server, sessionsServer)
	defer sessionsServer.Shutdown()

	// Create the daemon listener and defer its closure.
	listener, err := daemon.NewListener()
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

	// Wait for termination from a signal, the server, or the daemon server. We
	// treat daemon termination as a non-error.
	select {
	case sig := <-signalTermination:
		return errors.Errorf("terminated by signal: %s", sig)
	case <-daemonServer.Termination:
		return nil
	case err = <-serverErrors:
		return errors.Wrap(err, "premature server termination")
	}
}

var daemonRunCommand = &cobra.Command{
	Use:    "run",
	Short:  "Runs the Mutagen daemon",
	Run:    cmd.Mainify(daemonRunMain),
	Hidden: true,
}

var daemonRunConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := daemonRunCommand.Flags()

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&daemonRunConfiguration.help, "help", "h", false, "Show help information")
}
