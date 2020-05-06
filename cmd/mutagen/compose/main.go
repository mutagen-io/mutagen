package compose

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/fatih/color"
)

func rootMain(_ *cobra.Command, rawArguments []string) error {
	// Perform a basic command line parsing.
	arguments, err := parseArguments(rawArguments)
	if err != nil {
		return fmt.Errorf("unable to parse command line options: %w", err)
	}

	// Compute the effective project directory.
	projectDirectory := "."
	if arguments.projectDirectory != nil {
		projectDirectory = *arguments.projectDirectory
	}

	// Load environment variables from file (if one exists).
	environmentFileName := ".env"
	if arguments.environmentFile != nil {
		environmentFileName = *arguments.environmentFile
	}
	environmentFilePath := filepath.Join(projectDirectory, environmentFileName)
	fileEnvironment, err := environmentFromFile(environmentFilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("unable to load environment file (%s): %w", environmentFilePath, err)
	}

	// Load environment variables from the OS.
	osEnvironment := environmentFromOS()

	// Compute the effective environment.
	environment := make(map[string]string)
	for k, v := range fileEnvironment {
		environment[k] = v
	}
	for k, v := range osEnvironment {
		environment[k] = v
	}

	// TODO: Implement configuration loading and translation.

	// TODO: Implement command hooks.

	// Run Docker Compose. If it starts but fails, then we can assume that it
	// printed its own failure information and thus simply forward its exit
	// code. Other failure modes should be reported directly.
	// TODO: Use translated files here.
	if err := runCompose(arguments); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		} else {
			return fmt.Errorf("unable to run Docker Compose: %w", err)
		}
	}

	// Success.
	return nil
}

var RootCommand = &cobra.Command{
	Use:                "compose",
	Short:              "Run Docker Compose with Mutagen enhancements",
	RunE:               rootMain,
	SilenceUsage:       true,
	DisableFlagParsing: true,
}

func init() {
	// Mark the command as experimental.
	RootCommand.Short = RootCommand.Short + color.YellowString(" [Experimental]")
}
