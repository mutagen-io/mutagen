package main

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	promptpkg "github.com/havoc-io/mutagen/pkg/prompt"
	sessionsvcpkg "github.com/havoc-io/mutagen/pkg/service/session"
	sessionpkg "github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/sync"
	"github.com/havoc-io/mutagen/pkg/url"
)

func createMain(command *cobra.Command, arguments []string) error {
	// Validate, extract, and parse URLs.
	if len(arguments) != 2 {
		return errors.New("invalid number of endpoint URLs provided")
	}
	alpha, err := url.Parse(arguments[0], true)
	if err != nil {
		return errors.Wrap(err, "unable to parse alpha URL")
	}
	beta, err := url.Parse(arguments[1], false)
	if err != nil {
		return errors.Wrap(err, "unable to parse beta URL")
	}

	// If either URL is a local path, make sure it's normalized.
	if alpha.Protocol == url.Protocol_Local {
		if alphaPath, err := filesystem.Normalize(alpha.Path); err != nil {
			return errors.Wrap(err, "unable to normalize alpha path")
		} else {
			alpha.Path = alphaPath
		}
	}
	if beta.Protocol == url.Protocol_Local {
		if betaPath, err := filesystem.Normalize(beta.Path); err != nil {
			return errors.Wrap(err, "unable to normalize beta path")
		} else {
			beta.Path = betaPath
		}
	}

	// Validate and convert the symlink mode specification.
	var symlinkMode sync.SymlinkMode
	if createConfiguration.symlinkMode != "" {
		if err := symlinkMode.UnmarshalText([]byte(createConfiguration.symlinkMode)); err != nil {
			return errors.Wrap(err, "unable to parse symlink mode")
		}
	}

	// Validate and convert the watch mode specification.
	var watchMode filesystem.WatchMode
	if createConfiguration.watchMode != "" {
		if err := watchMode.UnmarshalText([]byte(createConfiguration.watchMode)); err != nil {
			return errors.Wrap(err, "unable to parse watch mode")
		}
	}

	// There's no need to validate the watch polling interval - any uint32 value
	// is valid.

	// We don't need to validate ignores here, that will happen on the session
	// service, so we'll save ourselves the time.

	// Validate and convert the VCS ignore mode specification.
	var ignoreVCSMode sync.IgnoreVCSMode
	if createConfiguration.ignoreVCS && createConfiguration.noIgnoreVCS {
		return errors.New("conflicting VCS ignore behavior specified")
	} else if createConfiguration.ignoreVCS {
		ignoreVCSMode = sync.IgnoreVCSMode_IgnoreVCS
	} else if createConfiguration.noIgnoreVCS {
		ignoreVCSMode = sync.IgnoreVCSMode_PropagateVCS
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := createDaemonClientConnection()
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := sessionsvcpkg.NewSessionsClient(daemonConnection)

	// Invoke the session create method. The stream will close when the
	// associated context is cancelled.
	createContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := sessionService.Create(createContext)
	if err != nil {
		return errors.Wrap(peelAwayRPCErrorLayer(err), "unable to invoke create")
	}

	var alphaWinsOnConflict, betaWinsOnConflict bool

	if createConfiguration.conflictWinner == "alpha" {
		alphaWinsOnConflict = true
		betaWinsOnConflict = false
	} else if createConfiguration.conflictWinner == "beta" {
		alphaWinsOnConflict = false
		betaWinsOnConflict = true
	}

	// Send the initial request.
	request := &sessionsvcpkg.CreateRequest{
		Alpha: alpha,
		Beta:  beta,
		Configuration: &sessionpkg.Configuration{
			SymlinkMode:          symlinkMode,
			WatchMode:            watchMode,
			WatchPollingInterval: createConfiguration.watchPollingInterval,
			Ignores:              createConfiguration.ignores,
			IgnoreVCSMode:        ignoreVCSMode,
			AlphaWinsOnConflict:  alphaWinsOnConflict,
			BetaWinsOnConflict:   betaWinsOnConflict,
		},
	}
	if err := stream.Send(request); err != nil {
		return errors.Wrap(peelAwayRPCErrorLayer(err), "unable to send create request")
	}

	// Create a status line printer and defer a break.
	statusLinePrinter := &cmd.StatusLinePrinter{}
	defer statusLinePrinter.BreakIfNonEmpty()

	// Receive and process responses until we're done.
	for {
		if response, err := stream.Recv(); err != nil {
			return errors.Wrap(peelAwayRPCErrorLayer(err), "create failed")
		} else if err = response.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid create response received")
		} else if response.Session != "" {
			statusLinePrinter.Print(fmt.Sprintf("Created session %s", response.Session))
			return nil
		} else if response.Message != "" {
			statusLinePrinter.Print(response.Message)
			if err := stream.Send(&sessionsvcpkg.CreateRequest{}); err != nil {
				return errors.Wrap(peelAwayRPCErrorLayer(err), "unable to send message response")
			}
		} else if response.Prompt != "" {
			statusLinePrinter.BreakIfNonEmpty()
			if response, err := promptpkg.PromptCommandLine(response.Prompt); err != nil {
				return errors.Wrap(err, "unable to perform prompting")
			} else if err = stream.Send(&sessionsvcpkg.CreateRequest{Response: response}); err != nil {
				return errors.Wrap(peelAwayRPCErrorLayer(err), "unable to send prompt response")
			}
		}
	}
}

var createCommand = &cobra.Command{
	Use:   "create <alpha> <beta>",
	Short: "Creates and starts a new synchronization session",
	Run:   cmd.Mainify(createMain),
}

var createConfiguration struct {
	help                 bool
	ignores              []string
	ignoreVCS            bool
	noIgnoreVCS          bool
	symlinkMode          string
	watchMode            string
	watchPollingInterval uint32
	conflictWinner       string
}

func init() {
	// Bind flags to configuration. We manually add help to override the default
	// message, but Cobra still implements it automatically.
	flags := createCommand.Flags()
	flags.BoolVarP(&createConfiguration.help, "help", "h", false, "Show help information")
	flags.StringSliceVarP(&createConfiguration.ignores, "ignore", "i", nil, "Specify ignore paths")
	flags.BoolVar(&createConfiguration.ignoreVCS, "ignore-vcs", false, "Ignore VCS directories")
	flags.BoolVar(&createConfiguration.noIgnoreVCS, "no-ignore-vcs", false, "Propagate VCS directories")
	flags.StringVar(&createConfiguration.symlinkMode, "symlink-mode", "", "Specify symlink mode (ignore|portable|posix-raw)")
	flags.StringVar(&createConfiguration.watchMode, "watch-mode", "", "Specify watch mode (portable|force-poll)")
	flags.Uint32Var(&createConfiguration.watchPollingInterval, "watch-polling-interval", 0, "Specify watch polling interval in seconds")
	flags.StringVar(&createConfiguration.conflictWinner, "conflict-winner", "", "Specify which side wins on conflict (alpha|beta)")
}
