package compose

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// shorthandFileFlagMatcher matches shorthand flag specifications containing
	// file specifications at their end or as the next argument. This requires
	// supporting other shorthand flags which don't take an argument and which
	// may precede the file flag specification. Such flags are specified in the
	// bracket expression of the regular expression and may need to be updated
	// if Docker Compose's command line interface evolves.
	shorthandFileFlagMatcher = regexp.MustCompile(`^-[hv]*f`)

	// shorthandProjectNameFlagMatcher matches shorthand flag specifications
	// containing project name specifications at their end or as the next
	// argument. The design is the same as that of shorthandFileFlagMatcher.
	shorthandProjectNameFlagMatcher = regexp.MustCompile(`^-[hv]*p`)
)

// arguments stores the components of the (partially) parsed Docker Compose
// command line arguments and flags.
type arguments struct {
	// files are any configuration files explicitly specified using the -f or
	// --file flags, in order of specification.
	files []string
	// preCommandArguments are those arguments and flags preceeding the command
	// specification (if any). It will have file specifications filtered out. If
	// a shorthand file specification is combined with other nullary shorthand
	// flags, then the file specification will be removed from that clustering
	// of flags but the remainder of the cluster will be included. No other
	// flags are filtered out, including those whose value may be extracted.
	preCommandArguments []string
	// command is the command specification (if any).
	command *string
	// postCommandArguments are those arguments and flags following the command
	// specification (if any). If a command was not specified, then this slice
	// will be empty.
	postCommandArguments []string
	// projectName is the project name specified by the -p or --project-name
	// flags, if any.
	projectName *string
	// projectDirectory is the project directory specified by the
	// --project-directory flag, if any.
	projectDirectory *string
	// environmentFile is the environment variables file specified by the
	// --env-file flag, if any.
	environmentFile *string
}

// reconstitute recombines arguments into a single slice suitable for using with
// os/exec.Command.
func (a *arguments) reconstitute() []string {
	// Compute the result size.
	count := len(a.files) + len(a.preCommandArguments)
	if a.command != nil {
		count += 1 + len(a.postCommandArguments)
	}

	// If there are no arguments, then we can return an empty slice.
	if count == 0 {
		return nil
	}

	// Preallocate the result slice.
	result := make([]string, 0, count)

	// Populate the argument slice.
	for _, file := range a.files {
		result = append(result, fmt.Sprintf("--file=%s", file))
	}
	result = append(result, a.preCommandArguments...)
	if a.command != nil {
		result = append(result, *a.command)
		result = append(result, a.postCommandArguments...)
	}

	// Done.
	return result
}

// stringptr takes a string value and returns a pointer to that value. It is a
// utility function for extracting string pointers from loop variables. It is
// guaranteed to return a non-nil result.
func stringptr(value string) *string {
	return &value
}

// parseArguments performs a cursory parsing of Docker Compose arguments and
// flags. It creates a structure that provides access to various elements of the
// arguments. This structure also allows for reconstitution of the flags for the
// purposes of invoking Docker Compose. This function avoids any disruption to
// the behavior of Docker Compose's parsing, so we only filter out file
// specifications and we keep behavioral parity with the command line parsing
// library (docopt) used by Docker Compose.
func parseArguments(rawArguments []string) (*arguments, error) {
	// Set up state tracking.
	var files, preCommandArguments, postCommandArguments []string
	var command, projectName, projectDirectory, environmentFile *string
	var nextIsFile, nextIsProjectName, nextIsProjectDirectory, nextIsEnvironmentFile bool

	// Process arguments.
	for _, argument := range rawArguments {
		if nextIsFile {
			files = append(files, argument)
			nextIsFile = false
		} else if nextIsProjectName {
			projectName = stringptr(argument)
			nextIsProjectName = false
		} else if nextIsProjectDirectory {
			projectDirectory = stringptr(argument)
			nextIsProjectDirectory = false
		} else if nextIsEnvironmentFile {
			environmentFile = stringptr(argument)
			nextIsEnvironmentFile = false
		} else if command != nil {
			postCommandArguments = append(postCommandArguments, argument)
		} else if argument == "--file" {
			nextIsFile = true
		} else if strings.HasPrefix(argument, "--file=") {
			files = append(files, argument[7:])
		} else if argument == "--project-name" {
			nextIsProjectName = true
		} else if strings.HasPrefix(argument, "--project-name=") {
			files = append(files, argument[15:])
		} else if argument == "--project-directory" {
			nextIsProjectDirectory = true
		} else if strings.HasPrefix(argument, "--project-directory=") {
			projectDirectory = stringptr(argument[20:])
		} else if argument == "--env-file" {
			nextIsEnvironmentFile = true
		} else if strings.HasPrefix(argument, "--env-file=") {
			environmentFile = stringptr(argument[11:])
		} else if shorthand := shorthandFileFlagMatcher.FindString(argument); shorthand != "" {
			if len(shorthand) == len(argument) {
				nextIsFile = true
			} else {
				files = append(files, argument[len(shorthand):])
			}
			if shorthand != "-f" {
				preCommandArguments = append(preCommandArguments, argument[:len(shorthand)-1])
			}
		} else if shorthand = shorthandProjectNameFlagMatcher.FindString(argument); shorthand != "" {
			if len(shorthand) == len(argument) {
				nextIsProjectName = true
			} else {
				files = append(files, argument[len(shorthand):])
			}
			if shorthand != "-p" {
				preCommandArguments = append(preCommandArguments, argument[:len(shorthand)-1])
			}
		} else if strings.HasPrefix(argument, "-") && argument != "-" && argument != "--" {
			preCommandArguments = append(preCommandArguments, argument)
		} else {
			command = stringptr(argument)
		}
	}

	// Check for unexpected termination.
	if nextIsFile {
		return nil, errors.New("missing file specification")
	} else if nextIsProjectName {
		return nil, errors.New("missing project name specification")
	} else if nextIsProjectDirectory {
		return nil, errors.New("missing project directory specification")
	} else if nextIsEnvironmentFile {
		return nil, errors.New("missing environment file specification")
	}

	// Success.
	return &arguments{
		files:                files,
		preCommandArguments:  preCommandArguments,
		command:              command,
		postCommandArguments: postCommandArguments,
		projectName:          projectName,
		projectDirectory:     projectDirectory,
		environmentFile:      environmentFile,
	}, nil
}
