package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
)

// generateMain is the entry point for the generate command.
func generateMain(_ *cobra.Command, _ []string) error {
	// Generate a Bash completion script, if requested.
	if generateConfiguration.bashCompletionScript != "" {
		if err := rootCommand.GenBashCompletionFile(generateConfiguration.bashCompletionScript); err != nil {
			return fmt.Errorf("unable to generate Bash completion script: %w", err)
		}
	}

	// Generate a fish completion script, if requested.
	if generateConfiguration.fishCompletionScript != "" {
		if err := rootCommand.GenFishCompletionFile(generateConfiguration.fishCompletionScript, true); err != nil {
			return fmt.Errorf("unable to generate fish completion script: %w", err)
		}
	}

	// Generate a PowerShell completion script, if requested.
	if generateConfiguration.powerShellCompletionScript != "" {
		if err := rootCommand.GenPowerShellCompletionFile(generateConfiguration.powerShellCompletionScript); err != nil {
			return fmt.Errorf("unable to generate PowerShell completion script: %w", err)
		}
	}

	// Generate a Zsh completion script, if requested.
	if generateConfiguration.zshCompletionScript != "" {
		if err := rootCommand.GenZshCompletionFile(generateConfiguration.zshCompletionScript); err != nil {
			return fmt.Errorf("unable to generate Zsh completion script: %w", err)
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
	// Bash completion script.
	bashCompletionScript string
	// fishCompletionScript indicates the path, if any, at which to generate the
	// fish completion script.
	fishCompletionScript string
	// powerShellCompletionScript indicates the path, if any, at which to
	// generate the PowerShell completion script.
	powerShellCompletionScript string
	// zshCompletionScript indicates the path, if any, at which to generate the
	// Zsh completion script.
	zshCompletionScript string
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
	flags.StringVar(&generateConfiguration.bashCompletionScript, "bash-completion-script", "", "Specify the Bash completion script output path")
	flags.StringVar(&generateConfiguration.fishCompletionScript, "fish-completion-script", "", "Specify the fish completion script output path")
	flags.StringVar(&generateConfiguration.powerShellCompletionScript, "powershell-completion-script", "", "Specify the PowerShell completion script output path")
	flags.StringVar(&generateConfiguration.zshCompletionScript, "zsh-completion-script", "", "Specify the Zsh completion script output path")
}
