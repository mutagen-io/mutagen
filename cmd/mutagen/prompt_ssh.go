package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/pkg/errors"

	promptpkg "github.com/havoc-io/mutagen/pkg/prompt"
	promptsvcpkg "github.com/havoc-io/mutagen/pkg/prompt/service"
	"github.com/havoc-io/mutagen/pkg/ssh"
)

func promptSSH(arguments []string) error {
	// Extract prompt.
	if len(arguments) != 1 {
		return errors.New("invalid number of arguments")
	}
	prompt := arguments[0]

	// Extract environment parameters.
	prompter := os.Getenv(ssh.PrompterEnvironmentVariable)
	if prompter == "" {
		return errors.New("no prompter specified")
	}
	messageBase64 := os.Getenv(ssh.PrompterMessageBase64EnvironmentVariable)
	if messageBase64 == "" {
		return errors.New("no message specified")
	}
	messageBytes, err := base64.StdEncoding.DecodeString(messageBase64)
	if err != nil {
		return errors.New("unable to decode message")
	}
	message := string(messageBytes)

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection()
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a prompt service client.
	promptService := promptsvcpkg.NewPromptClient(daemonConnection)

	// Invoke prompt.
	request := &promptsvcpkg.PromptRequest{
		Prompter: prompter,
		Prompt: &promptpkg.Prompt{
			Message: message,
			Prompt:  prompt,
		},
	}
	response, err := promptService.Prompt(context.Background(), request)
	if err != nil {
		return errors.Wrap(err, "unable to invoke prompt")
	}

	// Print the response.
	fmt.Println(response.Response)

	// Success.
	return nil
}
