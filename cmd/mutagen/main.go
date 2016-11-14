package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/texttheater/golang-levenshtein/levenshtein"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/environment"
	"github.com/havoc-io/mutagen/grpcutil"
	"github.com/havoc-io/mutagen/ssh"
)

func init() {
	// Squelch gRPC, because it thinks it owns standard error and vomits out
	// every internal diagnostic message.
	grpcutil.Squelch()
}

// usage provides help information for the main Mutagen entry point.
var usage = `usage: mutagen [-V|--version] [-h|--help] [-l|--legal] <command> [<args>]

Supported commands include:

    start           Start a new synchronization session
    list            List current synchronization sessions
    pause           Pause a synchronization session
    resume          Resume a synchronization session
    stop            Stop and remove a synchronization session
    daemon          Control the synchronization daemon lifecycle

To see help for a particular command, use 'mutagen <command> --help'.
`

// handlers maps command names to their handlers.
var handlers = map[string]func([]string){
	"list":   listMain,
	"start":  startMain,
	"pause":  pauseMain,
	"resume": resumeMain,
	"stop":   stopMain,
	"daemon": daemonMain,
}

// maximumCommandDistance specifies the maximum Levenshtein distance allowed for
// commands to be considered a match for suggestions.
const maximumCommandDistance = 4

func main() {
	// We have to do some manual argument parsing in here for command dispatch,
	// because none of the CLI parsing libraries provide a decent mechanism for
	// ensuring positional arguments appear before flags.

	// Extract arguments, sans program name
	arguments := os.Args[1:]
	nArguments := len(arguments)

	// Check if a prompting environment is set. If so, treat this as a prompt
	// request.
	if _, ok := environment.Current[ssh.PrompterEnvironmentVariable]; ok {
		promptMain(arguments)
		return
	}

	// Verify that there are arguments, otherwise print help and exit
	if nArguments == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	// Split up the arguments. We treat the first argument that doesn't start
	// with '-' as the command name, and all subsequent arguments as belonging
	// to that command.
	var command string
	var commandArguments []string
	for i := 0; i < nArguments; i++ {
		if arguments[i][0] != '-' {
			command = arguments[i]
			commandArguments = arguments[i+1:]
			arguments = arguments[:i]
			break
		}
	}

	// Parse and handle main entry point flags.
	flagSet := cmd.NewFlagSet("mutagen", usage, nil)
	version := flagSet.BoolP("version", "V", false, "")
	legal := flagSet.BoolP("legal", "l", false, "")
	flagSet.ParseOrDie(arguments)
	if *version {
		fmt.Println(mutagen.Version)
		return
	} else if *legal {
		fmt.Print(mutagen.LegalNotice)
		return
	}

	// If we haven't exited, then attempt to dispatch the command. The handler
	// may exit the program, but in case it doesn't we'll assume a successful
	// exit. We know that command will be non-empty at this point because there
	// were a non-0 number of arguments and there were no flags specified (if
	// there were flags specified, they would either have errored (because they
	// were incorrect) or exited (because that's what all of them do)).
	if handler, ok := handlers[command]; ok {
		handler(commandArguments)
		return
	}

	// If we couldn't dispatch, the command name is invalid.
	cmd.Error(errors.Errorf("unknown command: %s", command))

	// Try to find similar subcommands in case the user made a typo.
	var matches []string
	for name := range handlers {
		editDistance := levenshtein.DistanceForStrings(
			[]rune(command),
			[]rune(name),
			levenshtein.DefaultOptions,
		)
		if editDistance <= maximumCommandDistance {
			matches = append(matches, name)
		}
	}

	// Print similar subcommands, if any.
	if len(matches) > 0 {
		fmt.Fprintln(os.Stderr, "\nSimilar commands:")
		for _, match := range matches {
			fmt.Fprintf(os.Stderr, "\t%s\n", match)
		}
	}

	// Bail.
	os.Exit(1)
}
