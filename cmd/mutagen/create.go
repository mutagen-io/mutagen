package main

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/daemon"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/rpc"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/url"
)

func createMain(command *cobra.Command, arguments []string) {
	// Validate, extract, and parse URLs.
	if len(arguments) != 2 {
		cmd.Fatal(errors.New("invalid number of endpoint URLs provided"))
	}
	alpha, err := url.Parse(arguments[0])
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to parse alpha URL"))
	}
	beta, err := url.Parse(arguments[1])
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to parse beta URL"))
	}

	// If either URL is a local path, make sure it's normalized.
	if alpha.Protocol == url.Protocol_Local {
		if alphaPath, err := filesystem.Normalize(alpha.Path); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to normalize alpha path"))
		} else {
			alpha.Path = alphaPath
		}
	}
	if beta.Protocol == url.Protocol_Local {
		if betaPath, err := filesystem.Normalize(beta.Path); err != nil {
			cmd.Fatal(errors.Wrap(err, "unable to normalize beta path"))
		} else {
			beta.Path = betaPath
		}
	}

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the session creation method and ensure the resulting stream is
	// closed when we're done.
	stream, err := daemonClient.Invoke(sessionpkg.MethodCreate)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to invoke session creation"))
	}
	defer stream.Close()

	// Send the initial request.
	request := sessionpkg.CreateRequest{
		Alpha:   alpha,
		Beta:    beta,
		Ignores: createConfiguration.ignores,
	}
	if err := stream.Send(request); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to send creation request"))
	}

	// Handle authentication challenges.
	if err := handlePromptRequests(stream); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to handle prompt requests"))
	}

	// Receive the create response.
	var response sessionpkg.CreateResponse
	if err := stream.Receive(&response); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to receive create response"))
	}

	// Print the session identifier.
	fmt.Println("Created session", response.Session)
}

var createCommand = &cobra.Command{
	Use:   "create <alpha> <beta>",
	Short: "Creates and starts a new synchronization session",
	Run:   createMain,
}

var createConfiguration struct {
	help    bool
	ignores []string
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := createCommand.Flags()
	flags.BoolVarP(&createConfiguration.help, "help", "h", false, "Show help information")
	flags.StringSliceVarP(&createConfiguration.ignores, "ignore", "i", nil, "Specify ignore paths")
}
