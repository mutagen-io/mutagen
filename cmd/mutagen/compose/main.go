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

// stringptr takes a string value and returns a pointer to that value. It is a
// utility function for extracting string pointers from loop variables. It is
// guaranteed to return a non-nil result.
func stringptr(value string) *string {
	return &value
}

// shorthandFileFlagMatcher matches shorthand flag specifications containing
// file specifications at their end or as the next argument. This requires
// supporting other shorthand flags which don't take an argument and which may
// precede the file flag specification. Such flags are specified in the bracket
// expression of the regular expression and may need to be updated if Docker
// Compose's command line interface evolves.
var shorthandFileFlagMatcher = regexp.MustCompile(`^-[hv]*f`)

// runCompose invokes Docker Compose with the specified arguments.
func runCompose(files, preCommandArguments []string, command *string, postCommandArguments []string) error {
	// Preallocate the argument slice.
	argumentCount := len(files) + len(preCommandArguments)
	if command != nil {
		argumentCount += 1 + len(postCommandArguments)
	}
	arguments := make([]string, 0, argumentCount)

	// Populate the argument slice.
	for _, file := range files {
		arguments = append(arguments, fmt.Sprintf("--file=%s", file))
	}
	arguments = append(arguments, preCommandArguments...)
	if command != nil {
		arguments = append(arguments, *command)
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
	// Parse the command line to extract file specifications, project directory
	// specifications, environment variable file specifications, and the command
	// name, if any. We want to avoid any disruption to the behavior of Docker
	// Compose's parsing, so we only filter out file specifications and we keep
	// behavioral parity with Docker Compose's parser (docopt) when it comes to
	// identifying the command name.
	var files, preCommandArguments, postCommandArguments []string
	var command, projectDirectory, envFile *string
	var nextIsFile, nextIsProjectDirectory, nextIsEnvFile bool
	for _, argument := range arguments {
		if nextIsFile {
			files = append(files, argument)
			nextIsFile = false
		} else if nextIsProjectDirectory {
			projectDirectory = stringptr(argument)
			nextIsProjectDirectory = false
		} else if nextIsEnvFile {
			envFile = stringptr(argument)
			nextIsEnvFile = false
		} else if command != nil {
			postCommandArguments = append(postCommandArguments, argument)
		} else if argument == "--file" {
			nextIsFile = true
		} else if strings.HasPrefix(argument, "--file=") {
			files = append(files, argument[7:])
		} else if argument == "--project-directory" {
			nextIsProjectDirectory = true
		} else if strings.HasPrefix(argument, "--project-directory=") {
			projectDirectory = stringptr(argument[20:])
		} else if argument == "--env-file" {
			nextIsEnvFile = true
		} else if strings.HasPrefix(argument, "--env-file=") {
			envFile = stringptr(argument[11:])
		} else if shorthand := shorthandFileFlagMatcher.FindString(argument); shorthand != "" {
			if len(shorthand) == len(argument) {
				nextIsFile = true
			} else {
				files = append(files, argument[len(shorthand):])
			}
			if shorthand != "-f" {
				preCommandArguments = append(preCommandArguments, argument[:len(shorthand)-1])
			}
		} else if strings.HasPrefix(argument, "-") && argument != "-" && argument != "--" {
			preCommandArguments = append(preCommandArguments, argument)
		} else {
			command = stringptr(argument)
		}
	}
	if nextIsFile {
		return errors.New("missing file specification")
	} else if nextIsProjectDirectory {
		return errors.New("missing project directory specification")
	} else if nextIsEnvFile {
		return errors.New("missing environment file specification")
	}

	// TODO: Implement project loading.
	_ = projectDirectory
	_ = envFile

	// TODO: Intercept special commands and implement custom handling.

	// Run Docker Compose. If it starts but fails, then we can assume that it
	// printed its own failure information and thus simply forward its exit
	// code. Other failure modes should be reported directly.
	// TODO: Use translated files here.
	if err := runCompose(files, preCommandArguments, command, postCommandArguments); err != nil {
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
