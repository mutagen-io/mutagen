package main

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
)

// generateMain is the entry point for the generate command.
func generateMain(_ *cobra.Command, _ []string) error {
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

// generateCommand is the generate command.
var generateCommand = &cobra.Command{
	Use:          "generate",
	Short:        "Generate various files",
	Args:         cmd.DisallowArguments,
	Hidden:       true,
	RunE:         generateMain,
	SilenceUsage: true,
}

// generateConfiguration stores configuration for the generate command.
var generateConfiguration struct {
	// help indicates whether or not to show help information and exit.
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
