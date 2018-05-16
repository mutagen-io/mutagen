package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/session/service"
)

func pauseMain(command *cobra.Command, arguments []string) {
	// Parse session specification.
	var sessionQueries []string
	if len(arguments) > 0 {
		if pauseConfiguration.all {
			cmd.Fatal(errors.New("-a/--all specified with specific sessions"))
		}
		sessionQueries = arguments
	} else if !pauseConfiguration.all {
		cmd.Fatal(errors.New("no sessions specified"))
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection()
	if err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to connect to daemon"))
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := sessionsvcpkg.NewSessionClient(daemonConnection)

	// Invoke pause.
	request := &sessionsvcpkg.PauseRequest{
		All:            len(sessionQueries) == 0,
		SessionQueries: sessionQueries,
	}
	if _, err := sessionService.Pause(context.Background(), request); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to invoke pause"))
	}
}

var pauseCommand = &cobra.Command{
	Use:   "pause [<session>...]",
	Short: "Pauses a synchronization session",
	Run:   pauseMain,
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
