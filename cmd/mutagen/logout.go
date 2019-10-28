package main

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/fatih/color"

	"github.com/mutagen-io/mutagen/pkg/mutagenio"
)

func logoutMain(command *cobra.Command, arguments []string) error {
	// Validate arguments.
	if len(arguments) != 0 {
		return errors.New("unexpected arguments")
	}

	// Perform the logout.
	return mutagenio.Logout()
}

var logoutCommand = &cobra.Command{
	Use:          "logout",
	Short:        "Log out from mutagen.io",
	Hidden:       true,
	RunE:         logoutMain,
	SilenceUsage: true,
}

var logoutConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Mark the command as experimental.
	logoutCommand.Short = logoutCommand.Short + color.YellowString(" [Experimental]")

	// Grab a handle for the command line flags.
	flags := logoutCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&logoutConfiguration.help, "help", "h", false, "Show help information")
}
