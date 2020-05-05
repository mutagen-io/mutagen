package compose

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fatih/color"
)

// shorthandFileFlagMatcher matches shorthand flag specifications containing
// file specifications at their end or as the next argument. This requires
// supporting other shorthand flags which don't take an argument and which may
// precede the file flag specification. Such flags are specified in the bracket
// expression of the regular expression and may need to be updated if Docker
// Compose's command line interface evolves.
var shorthandFileFlagMatcher = regexp.MustCompile(`^-[hv]*f`)

// runCompose invokes Docker Compose with the specified arguments.
func runCompose(command string, files, preCommandArguments, postCommandArguments []string, commandSet bool) error {
	// Build the command line specification for Docker Compose.
	argumentCount := len(files) + len(preCommandArguments)
	if commandSet {
		argumentCount += 1 + len(postCommandArguments)
	}
	arguments := make([]string, 0, argumentCount)
	for _, file := range files {
		arguments = append(arguments, fmt.Sprintf("--file=%s", file))
	}
	arguments = append(arguments, preCommandArguments...)
	if commandSet {
		arguments = append(arguments, command)
		arguments = append(arguments, postCommandArguments...)
	}

	// Set up the command invocation.
	compose := exec.Command("docker-compose", arguments...)
	compose.Stdin = os.Stdin
	compose.Stdout = os.Stdout
	compose.Stderr = os.Stderr

	// TODO: Figure out signal handling. See what Docker Compose handles itself.

	// Run Docker Compose.
	return compose.Run()
}

func rootMain(_ *cobra.Command, arguments []string) error {
	// Parse the command line to extract file specifications and the command
	// name, if any. We want to avoid any disruption to the behavior of Docker
	// Compose's parsing, so we only filter out file specifications and we keep
	// behavioral parity with Docker Compose's parser (docopt) when it comes to
	// identifying the command name.
	var files, preCommandArguments, postCommandArguments []string
	var command string
	var commandSet, nextIsFileSpec bool
	for _, argument := range arguments {
		if nextIsFileSpec {
			files = append(files, argument)
			nextIsFileSpec = false
		} else if commandSet {
			postCommandArguments = append(postCommandArguments, argument)
		} else if argument == "--file" {
			nextIsFileSpec = true
		} else if strings.HasPrefix(argument, "--file=") {
			files = append(files, argument[7:])
		} else if shorthand := shorthandFileFlagMatcher.FindString(argument); shorthand != "" {
			if len(shorthand) == len(argument) {
				nextIsFileSpec = true
			} else {
				files = append(files, argument[len(shorthand):])
			}
			if shorthand != "-f" {
				preCommandArguments = append(preCommandArguments, argument[:len(shorthand)-1])
			}
		} else if strings.HasPrefix(argument, "-") && argument != "-" && argument != "--" {
			preCommandArguments = append(preCommandArguments, argument)
		} else {
			command = argument
			commandSet = true
		}
	}
	if nextIsFileSpec {
		return errors.New("missing file specification")
	}

	// TODO: Load configuration files and perform translation.

	// TODO: Intercept special commands and implement custom handling.

	// Run Docker Compose. If it starts but fails, then we can assume that it
	// printed its own failure information and thus simply forward its exit
	// code. Other failure modes should be reported directly.
	// TODO: Use translated files here.
	if err := runCompose(command, files, preCommandArguments, postCommandArguments, commandSet); err != nil {
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
