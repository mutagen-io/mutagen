package main

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/havoc-io/mutagen/pkg/daemon"
	mgrpc "github.com/havoc-io/mutagen/pkg/grpc"
)

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
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(mgrpc.MaximumIPCMessageSize)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(mgrpc.MaximumIPCMessageSize)),
	)
}

func peelAwayRPCErrorLayer(err error) error {
	if status, ok := grpcstatus.FromError(err); !ok {
		return err
	} else {
		return errors.New(status.Message())
	}
}
