package main

import (
	"net"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"golang.org/x/net/context"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/process"
)

func dialDaemon() (*grpc.ClientConn, error) {
	return grpc.Dial(
		"",
		grpc.WithBlock(),
		grpc.WithDialer(func(_ string, timeout time.Duration) (net.Conn, error) {
			return daemon.DialTimeout(timeout)
		}),
		grpc.WithInsecure(),
		grpc.WithTimeout(1*time.Second),
	)
}

var daemonUsage = `usage: mutagen daemon [-h|--help] [-s|--stop]

Controls the lifecycle of the Mutagen daemon. The default behavior of this
command is to start the Mutagen daemon in the background. The command is
idempotent - a daemon instance is only created if one doesn't already exist.
`

func daemonMain(arguments []string) {
	// Parse and handle flags.
	flagSet := cmd.NewFlagSet("daemon", daemonUsage, nil)
	run := flagSet.BoolP("run", "r", false, "run the daemon server")
	stop := flagSet.BoolP("stop", "s", false, "stop any running daemon server")
	flagSet.ParseOrDie(arguments)

	// Check that options are sane.
	if *run && *stop {
		cmd.Fatal(errors.New("-r/--run with -s/--stop doesn't make sense"))
	}

	// If stopping is requested, try to send a termination request.
	if *stop {
		// Create a daemon client and defer its closure.
		conn, err := dialDaemon()
		if err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to connect to daemon"))
		}
		defer conn.Close()

		// Create a daemon service client.
		client := daemon.NewDaemonClient(conn)

		// Attempt to invoke termination. We don't check for errors, because the
		// daemon may terminate before it can send a response.
		client.Terminate(
			context.Background(),
			&daemon.TerminateRequest{},
			grpc.FailFast(true),
		)

		// Done.
		return
	}

	// Unless running (non-backgrounding) is requested, then we need to restart
	// in the background.
	if !*run {
		// Attempt to fork/execute the daemon.
		daemonProcess := &exec.Cmd{
			Path:        process.Current.ExecutablePath,
			Args:        []string{"mutagen", "daemon", "--run"},
			SysProcAttr: daemonProcessAttributes(),
		}
		if err := daemonProcess.Start(); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to fork daemon"))
		}

		// Done.
		return
	}

	// Attempt to acquire the daemon lock and defer its release. If there is a
	// crash, the lock will be released by the OS automatically, but on Windows
	// this may only happen after some unspecified period of time (though it
	// does seem to be basically instant).
	lock, err := daemon.AcquireLock()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to acquire daemon lock"))
	}
	defer lock.Unlock()

	// Create a gRPC server with the necessary services.
	server, daemonTermination, err := daemon.NewServer()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create daemon server"))
	}

	// Create the daemon listener and defer its closure.
	listener, err := daemon.NewListener()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create daemon listener"))
	}
	defer listener.Close()

	// Serve incoming connections in a separate Goroutine, watching for serving
	// failure (likely due to failure in the underlying listener).
	servingTermination := make(chan error, 1)
	go func() {
		servingTermination <- server.Serve(listener)
	}()

	// Wait for termination from a signal, the server, or the daemon server.
	termination := make(chan os.Signal, 1)
	signal.Notify(termination, cmd.TerminationSignals...)
	select {
	case <-termination:
	case <-daemonTermination:
	case <-servingTermination:
	}
}
