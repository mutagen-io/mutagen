package main

import (
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/environment"
	"github.com/havoc-io/mutagen/pkg/ssh"
)

func prompt(prompt string) error {
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

	// Create a daemon client and defer its closure.
	daemonClient, err := createDaemonClient()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to create daemon client"))
	}
	defer daemonClient.Close()

	// Invoke the SSH prompt method and ensure the resulting stream is closed
	// when we're done.
	stream, err := daemonClient.Invoke(ssh.MethodPrompt)
	if err != nil {
		return errors.Wrap(err, "unable to invoke SSH prompting")
	}
	defer stream.Close()

	// Send the prompt request.
	if err := stream.Send(ssh.PromptRequest{
		Prompter: prompter,
		Message:  message,
		Prompt:   prompt,
	}); err != nil {
		return errors.Wrap(err, "unable to send prompt request")
	}

	// Receive the prompt response.
	var response ssh.PromptResponse
	if err := stream.Receive(&response); err != nil {
		return errors.Wrap(err, "unable to receive prompt response")
	}

	// Print the response.
	fmt.Println(response.Response)

	// Success.
	return nil
}
