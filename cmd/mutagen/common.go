package main

import (
	"context"
	"net"
	"time"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
)

// mainify is a small utility that wraps a non-standard Cobra entry point (one
// returning an error) and generates a standard Cobra entry point. It's useful
// for entry points to be able to rely on defer-based cleanup, which doesn't
// occur if the entry point terminates the process. This method allows the entry
// point to indicate an error while still performing cleanup.
func mainify(entry func(*cobra.Command, []string) error) func(*cobra.Command, []string) {
	return func(command *cobra.Command, arguments []string) {
		if err := entry(command, arguments); err != nil {
			cmd.Fatal(err)
		}
	}
}

func daemonDialer(_ string, timeout time.Duration) (net.Conn, error) {
	return daemon.DialTimeout(timeout)
}

func createDaemonClientConnection() (*grpc.ClientConn, error) {
	// Create a context to timeout the dial.
	dialContext, cancel := context.WithTimeout(
		context.Background(),
		daemon.RecommendedDialTimeout,
	)
	defer cancel()

	// Perform dialing.
	return grpc.DialContext(
		dialContext,
		"",
		grpc.WithInsecure(),
		grpc.WithDialer(daemonDialer),
		grpc.WithBlock(),
	)
}
