package main

import (
	"context"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/grpcutil"
	promptpkg "github.com/havoc-io/mutagen/pkg/prompt"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/service/session"
	"github.com/havoc-io/mutagen/pkg/session"
)

func resumeMain(command *cobra.Command, arguments []string) error {
	// Create session selection specification.
	selection := &session.Selection{
		All:            resumeConfiguration.all,
		Specifications: arguments,
		LabelSelector:  resumeConfiguration.labelSelector,
	}
	if err := selection.EnsureValid(); err != nil {
		return errors.Wrap(err, "invalid session selection specification")
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection(true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := sessionsvcpkg.NewSessionsClient(daemonConnection)

	// Invoke the session resume method. The stream will close when the
	// associated context is cancelled.
	resumeContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := sessionService.Resume(resumeContext)
	if err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to invoke resume")
	}

	// Send the initial request.
	request := &sessionsvcpkg.ResumeRequest{
		Selection: selection,
	}
	if err := stream.Send(request); err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send resume request")
	}

	// Create a status line printer.
	statusLinePrinter := &cmd.StatusLinePrinter{}

	// Receive and process responses until we're done.
	for {
		if response, err := stream.Recv(); err != nil {
			statusLinePrinter.BreakIfNonEmpty()
			return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "resume failed")
		} else if err = response.EnsureValid(); err != nil {
			statusLinePrinter.BreakIfNonEmpty()
			return errors.Wrap(err, "invalid resume response received")
		} else if response.Message == "" && response.Prompt == "" {
			statusLinePrinter.Clear()
			return nil
		} else if response.Message != "" {
			statusLinePrinter.Print(response.Message)
			if err := stream.Send(&sessionsvcpkg.ResumeRequest{}); err != nil {
				statusLinePrinter.BreakIfNonEmpty()
				return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send message response")
			}
		} else if response.Prompt != "" {
			statusLinePrinter.BreakIfNonEmpty()
			if response, err := promptpkg.PromptCommandLine(response.Prompt); err != nil {
				return errors.Wrap(err, "unable to perform prompting")
			} else if err = stream.Send(&sessionsvcpkg.ResumeRequest{Response: response}); err != nil {
				return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send prompt response")
			}
		}
	}
}

var resumeCommand = &cobra.Command{
	Use:   "resume [<session>...]",
	Short: "Resumes a paused or disconnected synchronization session",
	Run:   cmd.Mainify(resumeMain),
}

var resumeConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
	// all indicates whether or not all sessions should be resumed.
	all bool
	// labelSelector encodes a label selector to be used in identifying which
	// sessions should be paused.
	labelSelector string
}

func init() {
	// Grab a handle for the command line flags.
	flags := resumeCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&resumeConfiguration.help, "help", "h", false, "Show help information")

	// Wire up resume flags.
	flags.BoolVarP(&resumeConfiguration.all, "all", "a", false, "Resume all sessions")
	flags.StringVar(&resumeConfiguration.labelSelector, "label-selector", "", "Resume sessions matching the specified label selector")
}
