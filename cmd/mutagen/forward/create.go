package forward

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/havoc-io/mutagen/cmd"
	"github.com/havoc-io/mutagen/cmd/mutagen/daemon"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	forwardingpkg "github.com/havoc-io/mutagen/pkg/forwarding"
	"github.com/havoc-io/mutagen/pkg/grpcutil"
	promptpkg "github.com/havoc-io/mutagen/pkg/prompt"
	"github.com/havoc-io/mutagen/pkg/selection"
	forwardingsvcpkg "github.com/havoc-io/mutagen/pkg/service/forwarding"
	"github.com/havoc-io/mutagen/pkg/url"
	forwardingurl "github.com/havoc-io/mutagen/pkg/url/forwarding"
)

func createMain(command *cobra.Command, arguments []string) error {
	// Validate, extract, and parse URLs.
	if len(arguments) != 2 {
		return errors.New("invalid number of endpoint URLs provided")
	}
	source, err := url.Parse(arguments[0], url.Kind_Forwarding, true)
	if err != nil {
		return errors.Wrap(err, "unable to parse source URL")
	}
	destination, err := url.Parse(arguments[1], url.Kind_Forwarding, false)
	if err != nil {
		return errors.Wrap(err, "unable to parse destination URL")
	}

	// If either URL is a local Unix domain socket path, make sure it's
	// normalized.
	if source.Protocol == url.Protocol_Local {
		if protocol, path, err := forwardingurl.Parse(source.Path); err != nil {
			return errors.Wrap(err, "unable to parse source forwarding endpoint URL")
		} else if protocol == "unix" {
			if normalized, err := filesystem.Normalize(path); err != nil {
				return errors.Wrap(err, "unable to normalize source forwarding endpoint socket path")
			} else {
				source.Path = fmt.Sprintf("%s:%s", protocol, normalized)
			}
		}
	}
	if destination.Protocol == url.Protocol_Local {
		if protocol, path, err := forwardingurl.Parse(destination.Path); err != nil {
			return errors.Wrap(err, "unable to parse destination forwarding endpoint URL")
		} else if protocol == "unix" {
			if normalized, err := filesystem.Normalize(path); err != nil {
				return errors.Wrap(err, "unable to normalize destination forwarding endpoint socket path")
			} else {
				destination.Path = fmt.Sprintf("%s:%s", protocol, normalized)
			}
		}
	}

	// Parse, validate, and record labels.
	var labels map[string]string
	if len(createConfiguration.labels) > 0 {
		labels = make(map[string]string, len(createConfiguration.labels))
	}
	for _, label := range createConfiguration.labels {
		components := strings.SplitN(label, "=", 2)
		var key, value string
		key = components[0]
		if len(components) == 2 {
			value = components[1]
		}
		if err := selection.EnsureLabelKeyValid(key); err != nil {
			return errors.Wrap(err, "invalid label key")
		} else if err := selection.EnsureLabelValueValid(value); err != nil {
			return errors.Wrap(err, "invalid label value")
		}
		labels[key] = value
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.CreateClientConnection(true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Create a session service client.
	sessionService := forwardingsvcpkg.NewForwardingClient(daemonConnection)

	// Invoke the session create method. The stream will close when the
	// associated context is cancelled.
	createContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := sessionService.Create(createContext)
	if err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to invoke create")
	}

	// Send the initial request.
	request := &forwardingsvcpkg.CreateRequest{
		Specification: &forwardingsvcpkg.CreationSpecification{
			Source:                   source,
			Destination:              destination,
			Configuration:            &forwardingpkg.Configuration{},
			ConfigurationSource:      &forwardingpkg.Configuration{},
			ConfigurationDestination: &forwardingpkg.Configuration{},
			Labels:                   labels,
		},
	}
	if err := stream.Send(request); err != nil {
		return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send create request")
	}

	// Create a status line printer and defer a break.
	statusLinePrinter := &cmd.StatusLinePrinter{}
	defer statusLinePrinter.BreakIfNonEmpty()

	// Receive and process responses until we're done.
	for {
		if response, err := stream.Recv(); err != nil {
			return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "create failed")
		} else if err = response.EnsureValid(); err != nil {
			return errors.Wrap(err, "invalid create response received")
		} else if response.Session != "" {
			statusLinePrinter.Print(fmt.Sprintf("Created session %s", response.Session))
			return nil
		} else if response.Message != "" {
			statusLinePrinter.Print(response.Message)
			if err := stream.Send(&forwardingsvcpkg.CreateRequest{}); err != nil {
				return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send message response")
			}
		} else if response.Prompt != "" {
			statusLinePrinter.BreakIfNonEmpty()
			if response, err := promptpkg.PromptCommandLine(response.Prompt); err != nil {
				return errors.Wrap(err, "unable to perform prompting")
			} else if err = stream.Send(&forwardingsvcpkg.CreateRequest{Response: response}); err != nil {
				return errors.Wrap(grpcutil.PeelAwayRPCErrorLayer(err), "unable to send prompt response")
			}
		}
	}
}

var createCommand = &cobra.Command{
	Use:          "create <source> <destination>",
	Short:        "Create and start a new forwarding session",
	RunE:         createMain,
	SilenceUsage: true,
}

var createConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
	// labels are the label specifications for the session.
	labels []string
}

func init() {
	// Grab a handle for the command line flags.
	flags := createCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&createConfiguration.help, "help", "h", false, "Show help information")

	// Wire up label flags.
	flags.StringSliceVarP(&createConfiguration.labels, "label", "l", nil, "Specify labels")
}
