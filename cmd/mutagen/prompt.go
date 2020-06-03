package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"

	promptingpkg "github.com/mutagen-io/mutagen/pkg/prompting"
	promptingsvc "github.com/mutagen-io/mutagen/pkg/service/prompting"
)

// promptMain is the entry point for prompting.
func promptMain(arguments []string) error {
	// Extract prompt.
	if len(arguments) != 1 {
		return errors.New("invalid number of arguments")
	}
	prompt := arguments[0]

	// Extract environment parameters.
	prompter := os.Getenv(promptingpkg.PrompterEnvironmentVariable)
	if prompter == "" {
		return errors.New("no prompter specified")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(false, true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a prompt service client.
	promptingService := promptingsvc.NewPromptingClient(daemonConnection)

	// Invoke prompt.
	request := &promptingsvc.PromptRequest{
		Prompter: prompter,
		Prompt:   prompt,
	}
	response, err := promptingService.Prompt(context.Background(), request)
	if err != nil {
		return errors.Wrap(err, "unable to invoke prompt")
	} else if err = response.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid prompt response")
	}

	// Print the response.
	fmt.Println(response.Response)

	// Success.
	return nil
}
