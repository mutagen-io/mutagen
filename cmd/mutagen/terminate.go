package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/session/service"
)

func terminateMain(command *cobra.Command, arguments []string) {
	// Parse session specification.
	var sessionQueries []string
	if len(arguments) > 0 {
		if terminateConfiguration.all {
			cmd.Fatal(errors.New("-a/--all specified with specific sessions"))
		}
		sessionQueries = arguments
	} else if !terminateConfiguration.all {
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

	// Invoke terminate.
	request := &sessionsvcpkg.TerminateRequest{
		All:            len(sessionQueries) == 0,
		SessionQueries: sessionQueries,
	}
	if _, err := sessionService.Terminate(context.Background(), request); err != nil {
		cmd.Fatal(errors.Wrap(err, "unable to invoke terminate"))
	}
}

var terminateCommand = &cobra.Command{
	Use:   "terminate [<session>...]",
	Short: "Permanently terminates a synchronization session",
	Run:   terminateMain,
}

var terminateConfiguration struct {
	all  bool
	help bool
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := terminateCommand.Flags()
	flags.BoolVarP(&terminateConfiguration.all, "all", "a", false, "Terminate all sessions")
	flags.BoolVarP(&terminateConfiguration.help, "help", "h", false, "Show help information")
}
