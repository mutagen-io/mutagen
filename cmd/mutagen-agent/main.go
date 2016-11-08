package main

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/cmd"
)

func main() {
	// TODO: Check if the agent is being invoked in install mode.

	// Write our version to standard out to indicate the agent has started. This
	// is necessary when invoking over SSH to indicate that the process has
	// started correctly. It also provides a simple sanity check.
	if err := mutagen.SendVersion(os.Stdout); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to write version"))
	}

	// Create a gRPC server.
	// HACK: We explicitly don't do defer server.Stop() because it'll try to
	// close the standard input/output connection, which is not supported (due
	// to deadlocking). See the documentation in the agent package
	// (stdioConn.Close) for more information.
	server := grpc.NewServer()

	// TODO: Register filesystem service.

	// TODO: Register endpoint service.

	// Start serving in a separate Goroutine, monitoring for exit.
	servingTermination := make(chan error, 1)
	go func() {
		servingTermination <- server.Serve(agent.NewStdioListener())
	}()

	// Wait for termination from a signal or the server.
	termination := make(chan os.Signal, 1)
	signal.Notify(termination, cmd.TerminationSignals...)
	select {
	case <-termination:
	case <-servingTermination:
	}
}
