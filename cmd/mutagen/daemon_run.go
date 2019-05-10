package main

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
	daemonsvc "github.com/havoc-io/mutagen/pkg/service/daemon"
	promptsvc "github.com/havoc-io/mutagen/pkg/service/prompt"
	sessionsvc "github.com/havoc-io/mutagen/pkg/service/session"
	"github.com/havoc-io/mutagen/pkg/session"
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

	// Create a session manager and defer its shutdown.
	sessionManager, err := session.NewManager()
	if err != nil {
		return errors.Wrap(err, "unable to create session manager")
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

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&daemonRunConfiguration.help, "help", "h", false, "Show help information")
}
