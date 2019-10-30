package tunnel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/pkg/encoding"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/tunneling"
)

func hostMain(command *cobra.Command, arguments []string) error {
	// Validate arguments.
	if len(arguments) == 0 {
		return errors.New("missing tunnel host parameters path")
	} else if len(arguments) != 1 {
		return errors.New("invalid number of arguments")
	}
	hostParametersPath := arguments[0]

	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Load host parameters.
	hostParameters := &tunneling.TunnelHostParameters{}
	if err := encoding.LoadAndUnmarshalProtobuf(hostParametersPath, hostParameters); err != nil {
		return fmt.Errorf("unable to load host parameters: %w", err)
	} else if err = hostParameters.EnsureValid(); err != nil {
		return fmt.Errorf("invalid host parameters: %w", err)
	}

	// Perform hosting in a background Goroutine and track unrecoverable errors.
	unrecoverableHostingErrors := make(chan error, 1)
	go func() {
		for {
			severity, err := tunneling.HostTunnel(
				context.Background(),
				logging.RootLogger,
				hostParameters,
			)
			switch severity {
			case tunneling.ErrorSeverityRecoverable:
				continue
			case tunneling.ErrorSeverityDelayedRecoverable:
				time.Sleep(tunneling.HostTunnelRetryDelayTime)
				continue
			case tunneling.ErrorSeverityUnrecoverable:
				unrecoverableHostingErrors <- err
				return
			default:
				panic("unhandled severity level")
			}
		}
	}()

	// Wait for a hosting error or termination signal.
	select {
	case sig := <-signalTermination:
		return fmt.Errorf("terminated by signal: %s", sig)
	case err := <-unrecoverableHostingErrors:
		return fmt.Errorf("unrecoverable hosting failure: %w", err)
	}
}

var hostCommand = &cobra.Command{
	Use:          "host <tunnel-host-parameters-path>",
	Short:        "Host a tunnel",
	Hidden:       true,
	RunE:         hostMain,
	SilenceUsage: true,
}

var hostConfiguration struct {
	// help indicates whether or not help information should be shown for the
	// command.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := hostCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&hostConfiguration.help, "help", "h", false, "Show help information")
}
