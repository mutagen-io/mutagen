package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	"github.com/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/cmd"
)

func init() {
	// Squelch gRPC, because it thinks it owns standard error and vomits out
	// every internal diagnostic message.
	grpclog.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags))
}

func main() {
	// TODO: Check if the agent is being invoked in install mode.

	// Write our version to standard out to indicate the agent has started. This
	// is necessary when invoking over SSH to indicate that the process has
	// started correctly. It also provides a simple sanity check.
	if err := mutagen.SendVersion(os.Stdout); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to write version"))
	}

	// Create a gRPC server.
	server := grpc.NewServer()

	// TODO: Register filesystem service.

	// TODO: Register endpoint service.

	// Create a faux connection on standard input/output.
	// HACK: We don't register either stream as a closer, because a Close call
	// can't preempt a blocking Read or Write call (and all standard
	// input/output Read/Write calls are blocking because there's not really a
	// reliable way for Go to poll on arbitrary file descriptors). We're okay
	// not to close them manually though, because we'll just exit the process.
	// If we did try to Close though (either directly or by invoking
	// server.Stop), and gRPC was blocking in a Read or Write, the Close would
	// block, potentially indefinitely.
	stdio, stdioTermination := agent.NewIOConn(os.Stdin, os.Stdout)

	// Create a one-shot listener and start serving on that listener. This
	// listener will error out after the first accept, but by that time the lone
	// standard input/output connection will have been accepted and its
	// processing will have started in a separate Goroutine.
	stdioListener := agent.NewOneShotListener(stdio)
	server.Serve(stdioListener)

	// Wait for termination from a signal or the server.
	termination := make(chan os.Signal, 1)
	signal.Notify(termination, cmd.TerminationSignals...)
	select {
	case <-termination:
	case <-stdioTermination:
	}
}
