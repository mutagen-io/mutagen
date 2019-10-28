package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/pkg/mutagenio"
	"github.com/mutagen-io/mutagen/pkg/prompt"
)

func loginMain(command *cobra.Command, arguments []string) error {
	// Validate arguments.
	if len(arguments) != 0 {
		return errors.New("unexpected arguments")
	}

	// Prompt for the API token.
	apiToken, err := prompt.PromptCommandLineWithResponseMode("Enter API token: ", prompt.ResponseModeMasked)
	if err != nil {
		return fmt.Errorf("unable to read API token: %w", err)
	}

	// Perform basic validation of the token.
	if scheme, err := mutagenio.ExtractTokenScheme(apiToken); err != nil {
		return fmt.Errorf("unable to extract token scheme: %w", err)
	} else if scheme != mutagenio.TokenSchemeAPI {
		return errors.New("incorrect token scheme")
	}

	// Perform the login.
	if err := mutagenio.Login(apiToken); err != nil {
		return err
	}

	// Success.
	return nil
}

var loginCommand = &cobra.Command{
	Use:          "login",
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
	// Mark the command as experimental.
	loginCommand.Short = loginCommand.Short + color.YellowString(" [Experimental]")

	// Grab a handle for the command line flags.
	flags := loginCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&loginConfiguration.help, "help", "h", false, "Show help information")
}
