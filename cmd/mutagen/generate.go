package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
)

func generateMain(command *cobra.Command, arguments []string) error {
	// Ensure that no arguments have been provided.
	if len(arguments) > 0 {
		return errors.New("this command does not accept arguments")
	}

	// Ensure that at least one flag has been specified.
	flagSpecified := generateConfiguration.bashCompletionScript != ""
	if !flagSpecified {
		return errors.New("no flags specified")
	}

	// Generate bash completion script, if requested.
	if generateConfiguration.bashCompletionScript != "" {
		if err := rootCommand.GenBashCompletionFile(generateConfiguration.bashCompletionScript); err != nil {
			return errors.Wrap(err, "unable to generate bash completion script")
		}
	}

	// Success.
	return nil
}

var generateCommand = &cobra.Command{
	Use:    "generate",
	Short:  "Generate various files",
	Run:    cmd.Mainify(generateMain),
	Hidden: true,
}

var generateConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
	// bashCompletionScript indicates the path, if any, at which to generate the
	// bash completion script.
	bashCompletionScript string
}

func init() {
	// Grab a handle for the command line flags.
	flags := generateCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&generateConfiguration.help, "help", "h", false, "Show help information")

	// Wire up file generation flags.
	flags.StringVar(&generateConfiguration.bashCompletionScript, "bash-completion-script", "", "Generate bash completion script")
}
