package main

import (
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"

	"google.golang.org/grpc"

	contextpkg "golang.org/x/net/context"

	"github.com/havoc-io/mutagen/agent"
	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/environment"
)

var promptUsage = `usage: mutagen <prompt>
`

func promptMain(arguments []string) {
	// Parse command line arguments.
	flagSet := cmd.NewFlagSet("prompt", promptUsage, []int{1})
	prompt := flagSet.ParseOrDie(arguments)[0]

	// Extract environment parameters.
	prompter := environment.Current[agent.PrompterEnvironmentVariable]
	if prompter == "" {
		cmd.Fatal(errors.New("no prompter specified"))
	}
	contextBase64 := environment.Current[agent.PrompterContextBase64EnvironmentVariable]
	if contextBase64 == "" {
		cmd.Fatal(errors.New("no context specified"))
	}
	contextBytes, err := base64.StdEncoding.DecodeString(contextBase64)
	if err != nil {
		cmd.Fatal(errors.New("unable to decode context"))
	}
	context := string(contextBytes)

	// Connect to the daemon.
	conn, err := dialDaemon()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to connect to daemon"))
	}

	// Create a prompt service client.
	promptClient := agent.NewPromptClient(conn)

	// Issue the prompt request.
	response, err := promptClient.Prompt(
		contextpkg.Background(),
		&agent.PromptRequest{
			Prompter: prompter,
			Context:  context,
			Prompt:   prompt,
		},
		grpc.FailFast(true),
	)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to complete prompt request"))
	}

	// Print the response.
	fmt.Println(response.Response)
}
