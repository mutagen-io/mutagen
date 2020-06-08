package tunnel

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"

	"github.com/golang/protobuf/proto"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"

	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/selection"
	promptingsvc "github.com/mutagen-io/mutagen/pkg/service/prompting"
	tunnelingsvc "github.com/mutagen-io/mutagen/pkg/service/tunneling"
	"github.com/mutagen-io/mutagen/pkg/tunneling"
)

// createWithSpecification performs a create operation using the provided daemon
// connection and tunnel specification. It returns the resulting tunnel hosting
// credentials and any error that occurred.
func createWithSpecification(
	daemonConnection *grpc.ClientConn,
	specification *tunnelingsvc.CreationSpecification,
) (*tunneling.TunnelHostCredentials, error) {
	// Initiate command line messaging. We use standard error as our status
	// output stream since we write tunnel host parameters to standard output.
	statusLinePrinter := &cmd.StatusLinePrinter{UseStandardError: true}
	promptingCtx, promptingCancel := context.WithCancel(context.Background())
	prompter, promptingErrors, err := promptingsvc.Host(
		promptingCtx, promptingsvc.NewPromptingClient(daemonConnection),
		&cmd.StatusLinePrompter{Printer: statusLinePrinter}, false,
	)
	if err != nil {
		promptingCancel()
		return nil, errors.Wrap(err, "unable to initiate prompting")
	}

	// Perform the create operation, cancel prompting, and handle errors.
	tunnelingService := tunnelingsvc.NewTunnelingClient(daemonConnection)
	request := &tunnelingsvc.CreateRequest{
		Prompter:      prompter,
		Specification: specification,
	}
	response, err := tunnelingService.Create(context.Background(), request)
	promptingCancel()
	<-promptingErrors
	if err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return nil, grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		statusLinePrinter.BreakIfNonEmpty()
		return nil, errors.Wrap(err, "invalid create response received")
	}

	// Success.
	statusLinePrinter.Clear()
	return response.HostCredentials, nil
}

// createMain is the entry point for the create command.
func createMain(_ *cobra.Command, _ []string) error {
	// Validate the name.
	if err := selection.EnsureNameValid(createConfiguration.name); err != nil {
		return errors.Wrap(err, "invalid tunnel name")
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

	// Create a default tunnel configuration.
	configuration := &tunneling.Configuration{}

	// Create the creation specification.
	specification := &tunnelingsvc.CreationSpecification{
		Configuration: configuration,
		Name:          createConfiguration.name,
		Labels:        labels,
		Paused:        createConfiguration.paused,
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return errors.Wrap(err, "unable to connect to daemon")
	}
	defer daemonConnection.Close()

	// Perform the create operation.
	hostCredentials, err := createWithSpecification(daemonConnection, specification)
	if err != nil {
		return err
	}

	// Print the tunnel identifier to standard error.
	fmt.Fprintln(os.Stderr, "Created tunnel", hostCredentials.Identifier)

	// Write the tunnel host credentials to standard output.
	encodedHostCredentials, err := proto.Marshal(hostCredentials)
	if err != nil {
		return errors.Wrap(err, "unable to encode host parameters")
	} else if _, err := os.Stdout.Write(encodedHostCredentials); err != nil {
		return errors.Wrap(err, "unable to write encoded host parameters")
	}

	// Success.
	return nil
}

// createCommand is the create command.
var createCommand = &cobra.Command{
	Use:          "create",
	Short:        "Create and start a new tunnel",
	Args:         cmd.DisallowArguments,
	RunE:         createMain,
	SilenceUsage: true,
}

// createConfiguration stores configuration for the create command.
var createConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// name is the name specification for the tunnel.
	name string
	// labels are the label specifications for the tunnel.
	labels []string
	// paused indicates whether or not to create the tunnel in a pre-paused
	// state.
	paused bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := createCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&createConfiguration.help, "help", "h", false, "Show help information")

	// Wire up name and label flags.
	flags.StringVarP(&createConfiguration.name, "name", "n", "", "Specify a name for the tunnel")
	flags.StringSliceVarP(&createConfiguration.labels, "label", "l", nil, "Specify labels")

	// Wire up paused flags.
	flags.BoolVarP(&createConfiguration.paused, "paused", "p", false, "Create the tunnel pre-paused")
}
