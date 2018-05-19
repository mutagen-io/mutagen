package main

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/daemon"
	daemonsvcpkg "github.com/havoc-io/mutagen/pkg/daemon/service"
	promptsvcpkg "github.com/havoc-io/mutagen/pkg/prompt/service"
	"github.com/havoc-io/mutagen/pkg/session"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/session/service"
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

	// Perform housekeeping.
	agent.Housekeep()
	session.HousekeepCaches()
	session.HousekeepStaging()

	// Create the gRPC server.
	server := grpc.NewServer()

	// Create and register the daemon service.
	daemonService := daemonsvcpkg.New()
	daemonsvcpkg.RegisterDaemonServer(server, daemonService)

	// Create and register the prompt service.
	promptService := promptsvcpkg.New()
	promptsvcpkg.RegisterPromptServer(server, promptService)

	// Create and register the session service and defer its shutdown. We want
	// to do a clean shutdown because we don't want to lose information
	// generated during a synchronization cycle.
	sessionService, err := sessionsvcpkg.New(promptService)
	if err != nil {
		return errors.Wrap(err, "unable to create session service")
	}
	sessionsvcpkg.RegisterSessionServer(server, sessionService)
	defer sessionService.Shutdown()

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
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)
	select {
	case sig := <-signalTermination:
		return errors.Errorf("terminated by signal: %s", sig)
	case <-daemonService.Termination:
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
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := daemonRunCommand.Flags()
	flags.BoolVarP(&daemonRunConfiguration.help, "help", "h", false, "Show help information")
}
