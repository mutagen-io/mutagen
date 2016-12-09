package main

import (
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"

	"golang.org/x/net/context"

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
	messageBase64 := environment.Current[ssh.PrompterMessageBase64EnvironmentVariable]
	if messageBase64 == "" {
		cmd.Fatal(errors.New("no message specified"))
	}
	messageBytes, err := base64.StdEncoding.DecodeString(messageBase64)
	if err != nil {
		cmd.Fatal(errors.New("unable to decode message"))
	}
	message := string(messageBytes)

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
		context.Background(),
		&ssh.PromptRequest{
			Prompter: prompter,
			Message:  message,
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
