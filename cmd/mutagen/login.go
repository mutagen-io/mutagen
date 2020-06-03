package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/mutagenio"
	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// loginMain is the entry point for the login command.
func loginMain(_ *cobra.Command, _ []string) error {
	// Prompt for the API token.
	apiToken, err := prompting.PromptCommandLineWithResponseMode("Enter API token: ", prompting.ResponseModeMasked)
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

// loginCommand is the login command.
var loginCommand = &cobra.Command{
	Use:          "login",
	Short:        "Log in to mutagen.io",
	Args:         cmd.DisallowArguments,
	RunE:         loginMain,
	SilenceUsage: true,
}

// loginConfiguration stores configuration for the login command.
var loginConfiguration struct {
	// help indicates whether or not to show help information and exit.
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
