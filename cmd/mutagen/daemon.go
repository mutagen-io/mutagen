package main

import (
	"os"
	"os/exec"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/daemon"
	"github.com/havoc-io/mutagen/pkg/process"
	"github.com/havoc-io/mutagen/pkg/rpc"
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/ssh"
)

func daemonMain(command *cobra.Command, arguments []string) {
	// If no commands were given, then print help information and bail. We don't
	// have to worry about warning about arguments being present here (which
	// would be incorrect usage) because arguments can't even reach this point
	// (they will be mistaken for subcommands and a error will be displayed).
	command.Help()
}

var daemonCommand = &cobra.Command{
	Use:   "daemon",
	Short: "Controls the Mutagen daemon lifecycle",
	Run:   daemonMain,
}

var daemonConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := daemonCommand.Flags()
	flags.BoolVarP(&daemonConfiguration.help, "help", "h", false, "Show help information")

	// Register commands. We do this here (rather than in individual init
	// functions) so that we can control the order. If registration isn't
	// supported on the platform, then we exclude those commands. For some
	// reason, AddCommand can't be invoked twice, so we can't add these commands
	// conditionally later.
	if daemon.RegistrationSupported {
		daemonCommand.AddCommand(
			daemonRunCommand,
			daemonStartCommand,
			daemonStopCommand,
			daemonRegisterCommand,
			daemonUnregisterCommand,
		)
	} else {
		daemonCommand.AddCommand(
			daemonRunCommand,
			daemonStartCommand,
			daemonStopCommand,
		)
	}
}

func daemonRunMain(command *cobra.Command, arguments []string) {
	// Validate arguments.
	if len(arguments) != 0 {
		cmd.Fatal(errors.New("unexpected arguments provided"))
	}

	// TODO: Do we eventually want to encapsulate the construction of the daemon
	// RPC server into the daemon package, much like we do with endpoints? It
	// becomes a bit difficult to do cleanly. Also, I want the ability to have
	// different processes host the daemon (e.g. a GUI). In those cases, we may
	// want to add additional services that wouldn't be present in the CLI
	// daemon. So I'll leave things the way they are for now, but I'd like to
	// keep thinking about this for the future. One easy thing we could do is
	// move the daemon lock into the daemon service (and add a corresponding
	// shutdown method to the daemon service).

	// Attempt to acquire the daemon lock and defer its release. If there is a
	// crash, the lock will be released by the OS automatically, but on Windows
	// this may only happen after some unspecified period of time (though it
	// does seem to be basically instant).
	lock, err := daemon.AcquireLock()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to acquire daemon lock"))
	}
	defer lock.Unlock()

	// Perform housekeeping.
	agent.Housekeep()
	session.HousekeepCaches()
	session.HousekeepStaging()

	// Create the RPC server.
	server := rpc.NewServer()

	// Create and register the daemon service.
	daemonService, daemonTermination := daemon.NewService()
	server.Register(daemonService)

	// Create and regsiter the SSH service.
	sshService := ssh.NewService()
	server.Register(sshService)

	// Create the and register the session service and defer its shutdown. We
	// want to do a clean shutdown because we don't want to information
	// generated during a synchronization cycle.
	sessionService, err := session.NewService(sshService)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create session service"))
	}
	server.Register(sessionService)
	defer sessionService.Shutdown()

	// Create the daemon listener and defer its closure.
	listener, err := daemon.NewListener()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create daemon listener"))
	}
	defer listener.Close()

	// Serve incoming connections in a separate Goroutine, watching for serving
	// failure.
	serverTermination := make(chan error, 1)
	go func() {
		serverTermination <- server.Serve(listener)
	}()

	// Wait for termination from a signal, the server, or the daemon server. We
	// treat daemon termination as a non-error.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)
	select {
	case sig := <-signalTermination:
		cmd.Fatal(errors.Errorf("terminated by signal: %s", sig))
	case <-daemonTermination:
		return
	case err = <-serverTermination:
		cmd.Fatal(errors.Wrap(err, "premature server termination"))
	}
}

var daemonRunCommand = &cobra.Command{
	Use:    "run",
	Short:  "Runs the Mutagen daemon",
	Run:    daemonRunMain,
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

func daemonStartMain(command *cobra.Command, arguments []string) {
	// Validate arguments.
	if len(arguments) != 0 {
		cmd.Fatal(errors.New("unexpected arguments provided"))
	}

	// Check if start has been disabled due to registration with the system.
	if allowed, err := daemon.StartStopAllowed(); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to determine if start is allowed"))
	} else if !allowed {
		cmd.Fatal(errors.New("manual start not allowed while daemon is registered with system"))
	}

	// Restart in the background.
	daemonProcess := &exec.Cmd{
		Path:        process.Current.ExecutablePath,
		Args:        []string{"mutagen", "daemon", "run"},
		SysProcAttr: daemonProcessAttributes,
	}
	if err := daemonProcess.Start(); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to fork daemon"))
	}
}

var daemonStartCommand = &cobra.Command{
	Use:   "start",
	Short: "Starts the Mutagen daemon if it's not already running",
	Run:   daemonStartMain,
}

var daemonStartConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := daemonStartCommand.Flags()
	flags.BoolVarP(&daemonStartConfiguration.help, "help", "h", false, "Show help information")
}

func daemonStopMain(command *cobra.Command, arguments []string) {
	// Validate arguments.
	if len(arguments) != 0 {
		cmd.Fatal(errors.New("unexpected arguments provided"))
	}

	// Check if stop has been disabled due to registration with the system.
	if allowed, err := daemon.StartStopAllowed(); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to determine if stop is allowed"))
	} else if !allowed {
		cmd.Fatal(errors.New("manual stop not allowed while daemon is registered with system"))
	}

	// Create a daemon client and defer its closure.
	daemonClient, err := createDaemonClient()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create daemon client"))
	}
	defer daemonClient.Close()

	// Invoke termination.
	stream, err := daemonClient.Invoke(daemon.MethodTerminate)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to invoke daemon termination"))
	}
	stream.Close()
}

var daemonStopCommand = &cobra.Command{
	Use:   "stop",
	Short: "Stops the Mutagen daemon if it's running",
	Run:   daemonStopMain,
}

var daemonStopConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := daemonStopCommand.Flags()
	flags.BoolVarP(&daemonStopConfiguration.help, "help", "h", false, "Show help information")
}

func daemonRegisterMain(command *cobra.Command, arguments []string) {
	// Validate arguments.
	if len(arguments) != 0 {
		cmd.Fatal(errors.New("unexpected arguments provided"))
	}

	// Attempt registration.
	if err := daemon.Register(); err != nil {
		cmd.Fatal(err)
	}
}

var daemonRegisterCommand = &cobra.Command{
	Use:   "register",
	Short: "Registers Mutagen to start as a per-user daemon on login",
	Run:   daemonRegisterMain,
}

var daemonRegisterConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := daemonRegisterCommand.Flags()
	flags.BoolVarP(&daemonRegisterConfiguration.help, "help", "h", false, "Show help information")
}

func daemonUnregisterMain(command *cobra.Command, arguments []string) {
	// Validate arguments.
	if len(arguments) != 0 {
		cmd.Fatal(errors.New("unexpected arguments provided"))
	}

	// Attempt deregistration.
	if err := daemon.Unregister(); err != nil {
		cmd.Fatal(err)
	}
}

var daemonUnregisterCommand = &cobra.Command{
	Use:   "unregister",
	Short: "Unregisters Mutagen as a per-user daemon",
	Run:   daemonUnregisterMain,
}

var daemonUnregisterConfiguration struct {
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := daemonUnregisterCommand.Flags()
	flags.BoolVarP(&daemonUnregisterConfiguration.help, "help", "h", false, "Show help information")
}
