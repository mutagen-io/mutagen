package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/session/service"
)

func pauseMain(command *cobra.Command, arguments []string) error {
	// Parse session specifications.
	var specifications []string
	if len(arguments) > 0 {
		if pauseConfiguration.all {
			return errors.New("-a/--all specified with specific sessions")
		}
		specifications = arguments
	} else if !pauseConfiguration.all {
		return errors.New("no sessions specified")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection()
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := sessionsvcpkg.NewSessionClient(daemonConnection)

	// Invoke pause.
	request := &sessionsvcpkg.PauseRequest{
		Specifications: specifications,
	}
	if _, err := sessionService.Pause(context.Background(), request); err != nil {
		return errors.Wrap(err, "unable to invoke pause")
	}

	// Success.
	return nil
}

var pauseCommand = &cobra.Command{
	Use:   "pause [<session>...]",
	Short: "Pauses a synchronization session",
	Run:   mainify(pauseMain),
}

var pauseConfiguration struct {
	all  bool
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := pauseCommand.Flags()
	flags.BoolVarP(&pauseConfiguration.all, "all", "a", false, "Pause all sessions")
	flags.BoolVarP(&pauseConfiguration.help, "help", "h", false, "Show help information")
}
