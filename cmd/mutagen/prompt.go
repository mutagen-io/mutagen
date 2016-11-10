package main

import (
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"

	contextpkg "golang.org/x/net/context"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/environment"
	"github.com/havoc-io/mutagen/ssh"
)

var promptUsage = `usage: mutagen <prompt>
`

func promptMain(arguments []string) {
	// Parse command line arguments.
	flagSet := cmd.NewFlagSet("prompt", promptUsage, []int{1})
	prompt := flagSet.ParseOrDie(arguments)[0]

	// Extract environment parameters.
	prompter := environment.Current[ssh.PrompterEnvironmentVariable]
	if prompter == "" {
		cmd.Fatal(errors.New("no prompter specified"))
	}
	contextBase64 := environment.Current[ssh.PrompterContextBase64EnvironmentVariable]
	if contextBase64 == "" {
		cmd.Fatal(errors.New("no context specified"))
	}
	contextBytes, err := base64.StdEncoding.DecodeString(contextBase64)
	if err != nil {
		cmd.Fatal(errors.New("unable to decode context"))
	}
	context := string(contextBytes)

	// Create a daemon client connection and defer its closure.
	daemonClientConnection, err := newDaemonClientConnection()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to connect to daemon"))
	}
	defer daemonClientConnection.Close()

	// Create a prompt service client.
	promptClient := ssh.NewPromptClient(daemonClientConnection)

	// Issue the prompt request.
	response, err := promptClient.Request(
		contextpkg.Background(),
		&ssh.PromptRequest{
			Prompter: prompter,
			Context:  context,
			Prompt:   prompt,
		},
		grpcCallFlags...,
	)
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to complete prompt request"))
	}

	// Print the response.
	fmt.Println(response.Response)
}
