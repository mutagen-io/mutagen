package main

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/pkg/mutagenio"
)

func loginMain(command *cobra.Command, arguments []string) error {
	// Validate and extract the API token.
	if len(arguments) == 0 {
		return errors.New("API token required")
	} else if len(arguments) != 1 {
		return errors.New("invalid number of arguments")
	}
	apiToken := arguments[0]
	if apiToken == "" {
		return errors.New("empty API token")
	}

	// Perform the login.
	if err := mutagenio.Login(apiToken); err != nil {
		return err
	}

	// Success.
	return nil
}

var loginCommand = &cobra.Command{
	Use:          "login <api-token>",
	Short:        "Log in to mutagen.io",
	Hidden:       true,
	RunE:         loginMain,
	SilenceUsage: true,
}

var loginConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := loginCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&loginConfiguration.help, "help", "h", false, "Show help information")
}
