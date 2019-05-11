package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"

	promptpkg "github.com/havoc-io/mutagen/pkg/prompt"
	promptsvcpkg "github.com/havoc-io/mutagen/pkg/service/prompt"
)

func promptMain(arguments []string) error {
	// Extract prompt.
	if len(arguments) != 1 {
		return errors.New("invalid number of arguments")
	}
	prompt := arguments[0]

	// Extract environment parameters.
	prompter := os.Getenv(promptpkg.PrompterEnvironmentVariable)
	if prompter == "" {
		return errors.New("no prompter specified")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection(true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a prompt service client.
	promptService := promptsvcpkg.NewPromptingClient(daemonConnection)

	// Invoke prompt.
	request := &promptsvcpkg.PromptRequest{
		Prompter: prompter,
		Prompt:   prompt,
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
