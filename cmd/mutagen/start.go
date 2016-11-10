package main

import (
	"io"
	"path/filepath"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	"golang.org/x/net/context"

	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/url"
	"github.com/havoc-io/mutagen/session"
)

var startUsage = `usage: mutagen start [-h|--help] <alpha> <beta>
`

func startMain(arguments []string) {
	// Parse and handle flags.
	flagSet := cmd.NewFlagSet("start", startUsage, []int{2})
	urls := flagSet.ParseOrDie(arguments)

	// Extract URLs.
	alpha := urls[0]
	beta := urls[1]

	// If either URL is a relative path, convert it to an absolute path.
	if url.Classify(alpha) == url.TypePath {
		if alphaAbs, err := filepath.Abs(alpha); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to make first path absolute"))
		} else {
			alpha = alphaAbs
		}
	}
	if url.Classify(beta) == url.TypePath {
		if betaAbs, err := filepath.Abs(beta); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to make second path absolute"))
		} else {
			beta = betaAbs
		}
	}

	// Create a daemon client and defer its closure.
	conn, err := dialDaemon()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to connect to daemon"))
	}
	defer conn.Close()

	// Create a session manager client.
	client := session.NewManagerClient(conn)

	// Create a start request.
	stream, err := client.Start(context.Background(), grpc.FailFast(true))
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to begin start request"))
	}

	// Send the initial request.
	if err := stream.Send(&session.StartRequest{Alpha: alpha, Beta: beta}); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to send start request"))
	}

	// Process prompt requests.
	for {
		// Read the next prompt request.
		request, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			cmd.Fatal(errors.Wrap(err, "unable to receive prompt request"))
		}

		// Perform prompting on the command line.
		response, err := agent.PromptCommandLine(
			request.Prompt.Context,
			request.Prompt.Prompt,
		)
		if err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to prompt"))
		}

		// Write the prompt response.
		if err := stream.Send(&session.StartRequest{Response: &agent.PromptResponse{response}}); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to send prompt request"))
		}
	}

	// TODO: Implement.
}
