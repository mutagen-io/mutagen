package main

import (
	"os"
	"os/exec"
	"os/signal"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/process"
	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/session"
	"github.com/havoc-io/mutagen/ssh"
)

var daemonUsage = `usage: mutagen daemon [-h|--help] [-s|--stop]

Controls the lifecycle of the Mutagen daemon. The default behavior of this
command is to start the Mutagen daemon in the background. The command is
idempotent - a daemon instance is only created if one doesn't already exist.
`

const (
	daemonMethodTerminate  = "daemon.Terminate"
	sshMethodPrompt        = "ssh.Prompt"
	sessionMethodCreate    = "session.Create"
	sessionMethodList      = "session.List"
	sessionMethodPause     = "session.Pause"
	sessionMethodResume    = "session.Resume"
	sessionMethodTerminate = "session.Terminate"
)

func daemonMain(arguments []string) error {
	// Parse flags.
	flagSet := cmd.NewFlagSet("daemon", daemonUsage, nil)
	run := flagSet.BoolP("run", "r", false, "run the daemon server")
	stop := flagSet.BoolP("stop", "s", false, "stop any running daemon server")
	flagSet.ParseOrDie(arguments)

	// Check that options are sane.
	if *run && *stop {
		return errors.New("-r/--run with -s/--stop doesn't make sense")
	}

	// If stopping is requested, try to send a termination request.
	if *stop {
		daemonClient := rpc.NewClient(daemon.NewOpener())
		stream, err := daemonClient.Invoke(daemonMethodTerminate)
		stream.Close()
		if err != nil {
			return errors.Wrap(err, "unable to invoke daemon termination")
		}
		return nil
	}

	// Unless running (non-backgrounding) is requested, then we need to restart
	// in the background.
	if !*run {
		daemonProcess := &exec.Cmd{
			Path:        process.Current.ExecutablePath,
			Args:        []string{"mutagen", "daemon", "--run"},
			SysProcAttr: daemonProcessAttributes,
		}
		if err := daemonProcess.Start(); err != nil {
			return errors.Wrap(err, "unable to fork daemon")
		}
		return nil
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

	// Create the daemon service.
	daemonService, daemonTermination := daemon.NewService()

	// Create the SSH service.
	sshService := ssh.NewService()

	// Create the session service and defer its shutdown. We want to do a clean
	// shutdown because we don't want to information generated during a
	// synchronization cycle.
	sessionService, err := session.NewService(sshService)
	if err != nil {
		return errors.Wrap(err, "unable to create session service")
	}
	defer sessionService.Shutdown()

	// Create the RPC server.
	server := rpc.NewServer(map[string]rpc.Handler{
		daemonMethodTerminate:  daemonService.Terminate,
		sshMethodPrompt:        sshService.Prompt,
		sessionMethodCreate:    sessionService.Create,
		sessionMethodList:      sessionService.List,
		sessionMethodPause:     sessionService.Pause,
		sessionMethodResume:    sessionService.Resume,
		sessionMethodTerminate: sessionService.Terminate,
	})

	// Create the daemon listener and defer its closure.
	listener, err := daemon.NewListener()
	if err != nil {
		return errors.Wrap(err, "unable to create daemon listener")
	}
	defer listener.Close()

	// Serve incoming connections in a separate Goroutine, watching for serving
	// failure (which will be due to the underlying listener).
	listenerTermination := make(chan error, 1)
	go func() {
		listenerTermination <- server.Serve(listener)
	}()

	// Wait for termination from a signal, the server, or the daemon server. We
	// only treat listener termination as an error.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)
	select {
	case <-signalTermination:
	case <-daemonTermination:
	case err = <-listenerTermination:
		return errors.Wrap(err, "premature listener termination")
	}

	// Success.
	return nil
}
