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

const (
	// hostCredentialsEnvironmentVariable is the name of the environment
	// variable that can be used to specified tunnel host credentials.
	hostCredentialsEnvironmentVariable = "MUTAGEN_TUNNEL_HOST_CREDENTIALS"
)

func hostMain(_ *cobra.Command, arguments []string) error {
	// Validate arguments and determine the path to the host credentials file.
	var hostCredentialsPath string
	if len(arguments) == 0 {
		if p := os.Getenv(hostCredentialsEnvironmentVariable); p != "" {
			hostCredentialsPath = p
		} else {
			return errors.New("missing tunnel host credentials path")
		}
	} else if len(arguments) == 1 {
		if os.Getenv(hostCredentialsEnvironmentVariable) != "" {
			return errors.New("tunnel host credentials path specified both in environment and on command line")
		} else {
			hostCredentialsPath = arguments[0]
		}
	} else {
		return errors.New("invalid number of arguments")
	}

	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Load host parameters.
	hostCredentials := &tunneling.TunnelHostCredentials{}
	if err := encoding.LoadAndUnmarshalProtobuf(hostCredentialsPath, hostCredentials); err != nil {
		return fmt.Errorf("unable to load host parameters: %w", err)
	} else if err = hostCredentials.EnsureValid(); err != nil {
		return fmt.Errorf("invalid host parameters: %w", err)
	}

	// Create a logger.
	logger := logging.RootLogger.Sublogger("hosting")

	// Perform hosting in a background Goroutine and track unrecoverable errors.
	unrecoverableHostingErrors := make(chan error, 1)
	go func() {
		for {
			severity, err := tunneling.HostTunnel(context.Background(), logger, hostCredentials)
			switch severity {
			case tunneling.ErrorSeverityRecoverable:
				logger.Info("hosting restart due to recoverable error:", err)
				continue
			case tunneling.ErrorSeverityDelayedRecoverable:
				logger.Info("delayed hosting restart due to recoverable error:", err)
				time.Sleep(tunneling.HostTunnelRetryDelayTime)
				continue
			case tunneling.ErrorSeverityUnrecoverable:
				logger.Info("hosting failed due to unrecoverable error:", err)
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
	Use:          "host <tunnel-host-credentials-path>",
	Short:        "Host a tunnel",
	RunE:         hostMain,
	SilenceUsage: true,
}

var hostConfiguration struct {
	// help indicates whether or not to show help information and exit.
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
