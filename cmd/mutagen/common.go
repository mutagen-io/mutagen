package main

import (
	"time"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/grpcutil"
	"github.com/havoc-io/mutagen/ssh"
)

const (
	daemonConnectionTimeout = 1 * time.Second
)

func newDaemonClientConnection() (*grpc.ClientConn, error) {
	// Connect to the daemon.
	connection, err := daemon.DialTimeout(daemonConnectionTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create IPC connection")
	}

	// Create a gRPC client that won't attempt re-dialing.
	return grpcutil.NewNonRedialingClientConnection(connection), nil
}

var grpcCallFlags = []grpc.CallOption{
	grpc.FailFast(true),
}

func performPrompts(client ssh.Prompt_RespondClient) error {
	// Close up the client once we're done.
	defer client.CloseSend()

	// Handle prompts until there's an error.
	for {
		// Receive the next request.
		request, err := client.Recv()
		if err != nil {
			return errors.Wrap(err, "unable to receive prompt")
		}

		// Perform the prompt.
		response, err := ssh.PromptCommandLine(request.Context, request.Prompt)
		if err != nil {
			return errors.Wrap(err, "unable to perform prompting")
		}

		// Send the response.
		if err := client.Send(&ssh.PromptResponse{Response: response}); err != nil {
			return errors.Wrap(err, "unable to send response")
		}
	}
}
