package main

import (
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/daemon"
	"github.com/havoc-io/mutagen/environment"
	"github.com/havoc-io/mutagen/rpc"
	"github.com/havoc-io/mutagen/ssh"
)

var promptUsage = `usage: mutagen <prompt>
`

func promptMain(arguments []string) error {
	// Parse command line arguments.
	flagSet := cmd.NewFlagSet("prompt", promptUsage, []int{1})
	prompt := flagSet.ParseOrDie(arguments)[0]

	// Extract environment parameters.
	prompter := environment.Current[ssh.PrompterEnvironmentVariable]
	if prompter == "" {
		return errors.New("no prompter specified")
	}
	messageBase64 := environment.Current[ssh.PrompterMessageBase64EnvironmentVariable]
	if messageBase64 == "" {
		return errors.New("no message specified")
	}
	messageBytes, err := base64.StdEncoding.DecodeString(messageBase64)
	if err != nil {
		return errors.New("unable to decode message")
	}
	message := string(messageBytes)

	// Create a daemon client.
	daemonClient := rpc.NewClient(daemon.NewOpener())

	// Invoke the SSH prompt method and ensure the resulting stream is closed
	// when we're done.
	stream, err := daemonClient.Invoke(ssh.MethodPrompt)
	if err != nil {
		return errors.Wrap(err, "unable to invoke SSH prompting")
	}
	defer stream.Close()

	// Send the prompt request and receive the response.
	var response ssh.PromptResponse
	if err := stream.Encode(ssh.PromptRequest{
		Prompter: prompter,
		Message:  message,
		Prompt:   prompt,
	}); err != nil {
		return errors.Wrap(err, "unable to send prompt request")
	} else if err := stream.Decode(&response); err != nil {
		return errors.Wrap(err, "unable to receive prompt response")
	}

	// Print the response.
	fmt.Println(response.Response)

	// Success.
	return nil
}
