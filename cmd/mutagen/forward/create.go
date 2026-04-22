package forward

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"

	"github.com/mutagen-io/mutagen/cmd"
	"github.com/mutagen-io/mutagen/cmd/mutagen/daemon"

	"github.com/mutagen-io/mutagen/pkg/configuration/global"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/selection"
	forwardingsvc "github.com/mutagen-io/mutagen/pkg/service/forwarding"
	promptingsvc "github.com/mutagen-io/mutagen/pkg/service/prompting"
	"github.com/mutagen-io/mutagen/pkg/url"
)

// loadAndValidateGlobalSynchronizationConfiguration loads a YAML-based global
// configuration, extracts the forwarding component, converts it to a Protocol
// Buffers session configuration, and validates it.
func loadAndValidateGlobalForwardingConfiguration(path string) (*forwarding.Configuration, error) {
	// Load the YAML configuration.
	yamlConfiguration, err := global.LoadConfiguration(path)
	if err != nil {
		return nil, err
	}

	// Convert the YAML configuration to a Protocol Buffers representation and
	// validate it.
	configuration := yamlConfiguration.Forwarding.Defaults.ToInternal()
	if err := configuration.EnsureValid(false); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Success.
	return configuration, nil
}

// CreateWithSpecification is an orchestration convenience method that performs
// a create operation using the provided daemon connection and session
// specification.
func CreateWithSpecification(
	daemonConnection *grpc.ClientConn,
	specification *forwardingsvc.CreationSpecification,
) (string, error) {
	// Initiate command line prompting.
	statusLinePrinter := &cmd.StatusLinePrinter{}
	promptingCtx, promptingCancel := context.WithCancel(context.Background())
	prompter, promptingErrors, err := promptingsvc.Host(
		promptingCtx, promptingsvc.NewPromptingClient(daemonConnection),
		&cmd.StatusLinePrompter{Printer: statusLinePrinter}, true,
	)
	if err != nil {
		promptingCancel()
		return "", fmt.Errorf("unable to initiate prompting: %w", err)
	}

	// Perform the create operation, cancel prompting, and handle errors.
	forwardingService := forwardingsvc.NewForwardingClient(daemonConnection)
	request := &forwardingsvc.CreateRequest{
		Prompter:      prompter,
		Specification: specification,
	}
	response, err := forwardingService.Create(context.Background(), request)
	promptingCancel()
	<-promptingErrors
	if err != nil {
		statusLinePrinter.BreakIfPopulated()
		return "", grpcutil.PeelAwayRPCErrorLayer(err)
	} else if err = response.EnsureValid(); err != nil {
		statusLinePrinter.BreakIfPopulated()
		return "", fmt.Errorf("invalid create response received: %w", err)
	}

	// Success.
	statusLinePrinter.Clear()
	return response.Session, nil
}

// createMain is the entry point for the create command.
func createMain(_ *cobra.Command, arguments []string) error {
	// Validate, extract, and parse URLs.
	if len(arguments) != 2 {
		return errors.New("invalid number of endpoint URLs provided")
	}
	source, err := url.Parse(arguments[0], url.Kind_Forwarding, true)
	if err != nil {
		return fmt.Errorf("unable to parse source URL: %w", err)
	}
	destination, err := url.Parse(arguments[1], url.Kind_Forwarding, false)
	if err != nil {
		return fmt.Errorf("unable to parse destination URL: %w", err)
	}

	if source.Protocol == url.Protocol_SSH {
		sshConfigPath := os.Getenv("MUTAGEN_SSH_CONFIG_SOURCE")
		if sshConfigPath == "" {
			sshConfigPath = os.Getenv("MUTAGEN_SSH_CONFIG")
		}
		if sshConfigPath != "" {
			if source.Parameters == nil {
				source.Parameters = make(map[string]string)
			}
			source.Parameters["ssh-config-path"] = sshConfigPath
		}
	}
	if destination.Protocol == url.Protocol_SSH {
		sshConfigPath := os.Getenv("MUTAGEN_SSH_CONFIG_DESTINATION")
		if sshConfigPath == "" {
			sshConfigPath = os.Getenv("MUTAGEN_SSH_CONFIG")
		}
		if sshConfigPath != "" {
			if destination.Parameters == nil {
				destination.Parameters = make(map[string]string)
			}
			destination.Parameters["ssh-config-path"] = sshConfigPath
		}
	}

	// Validate the name.
	if err := selection.EnsureNameValid(createConfiguration.name); err != nil {
		return fmt.Errorf("invalid session name: %w", err)
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
			return fmt.Errorf("invalid label key: %w", err)
		} else if err := selection.EnsureLabelValueValid(value); err != nil {
			return fmt.Errorf("invalid label value: %w", err)
		}
		labels[key] = value
	}

	// Create a default session configuration that will form the basis of our
	// cumulative configuration.
	configuration := &forwarding.Configuration{}

	// Unless disabled, attempt to load configuration from the global
	// configuration file and merge it into our cumulative configuration.
	if !createConfiguration.noGlobalConfiguration {
		// Compute the path to the global configuration file.
		globalConfigurationPath, err := global.ConfigurationPath()
		if err != nil {
			return fmt.Errorf("unable to compute path to global configuration file: %w", err)
		}

		// Attempt to load the file. We allow it to not exist.
		globalConfiguration, err := loadAndValidateGlobalForwardingConfiguration(globalConfigurationPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("unable to load global configuration: %w", err)
			}
		} else {
			configuration = forwarding.MergeConfigurations(configuration, globalConfiguration)
		}
	}

	// If additional default configuration files have been specified, then load
	// them and merge them into the cumulative configuration.
	for _, configurationFile := range createConfiguration.configurationFiles {
		if c, err := loadAndValidateGlobalForwardingConfiguration(configurationFile); err != nil {
			return fmt.Errorf("unable to load configuration file (%s): %w", configurationFile, err)
		} else {
			configuration = forwarding.MergeConfigurations(configuration, c)
		}
	}

	// Validate and convert socket overwrite mode specifications.
	var socketOverwriteMode, socketOverwriteModeSource, socketOverwriteModeDestination forwarding.SocketOverwriteMode
	if createConfiguration.socketOverwriteMode != "" {
		if err := socketOverwriteMode.UnmarshalText([]byte(createConfiguration.socketOverwriteMode)); err != nil {
			return fmt.Errorf("unable to socket overwrite mode: %w", err)
		}
	}
	if createConfiguration.socketOverwriteModeSource != "" {
		if err := socketOverwriteModeSource.UnmarshalText([]byte(createConfiguration.socketOverwriteModeSource)); err != nil {
			return fmt.Errorf("unable to socket overwrite mode for source: %w", err)
		}
	}
	if createConfiguration.socketOverwriteModeDestination != "" {
		if err := socketOverwriteModeDestination.UnmarshalText([]byte(createConfiguration.socketOverwriteModeDestination)); err != nil {
			return fmt.Errorf("unable to socket overwrite mode for destination: %w", err)
		}
	}

	// Validate socket owner specifications.
	if createConfiguration.socketOwner != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.socketOwner,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid socket ownership specification")
		}
	}
	if createConfiguration.socketOwnerSource != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.socketOwnerSource,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid socket ownership specification for source")
		}
	}
	if createConfiguration.socketOwnerDestination != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.socketOwnerDestination,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid socket ownership specification for destination")
		}
	}

	// Validate socket group specifications.
	if createConfiguration.socketGroup != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.socketGroup,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid socket group specification")
		}
	}
	if createConfiguration.socketGroupSource != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.socketGroupSource,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid socket group specification for source")
		}
	}
	if createConfiguration.socketGroupDestination != "" {
		if kind, _ := filesystem.ParseOwnershipIdentifier(
			createConfiguration.socketGroupDestination,
		); kind == filesystem.OwnershipIdentifierKindInvalid {
			return errors.New("invalid socket group specification for destination")
		}
	}

	// Validate and convert socket permission mode specifications.
	var socketPermissionMode, socketPermissionModeSource, socketPermissionModeDestination filesystem.Mode
	if createConfiguration.socketPermissionMode != "" {
		if err := socketPermissionMode.UnmarshalText([]byte(createConfiguration.socketPermissionMode)); err != nil {
			return fmt.Errorf("unable to parse socket permission mode: %w", err)
		}
	}
	if createConfiguration.socketPermissionModeSource != "" {
		if err := socketPermissionModeSource.UnmarshalText([]byte(createConfiguration.socketPermissionModeSource)); err != nil {
			return fmt.Errorf("unable to parse socket permission mode for source: %w", err)
		}
	}
	if createConfiguration.socketPermissionModeDestination != "" {
		if err := socketPermissionModeDestination.UnmarshalText([]byte(createConfiguration.socketPermissionModeDestination)); err != nil {
			return fmt.Errorf("unable to parse socket permission mode for destination: %w", err)
		}
	}

	// Create the command line configuration and merge it into our cumulative
	// configuration.
	configuration = forwarding.MergeConfigurations(configuration, &forwarding.Configuration{
		SocketOverwriteMode:  socketOverwriteMode,
		SocketOwner:          createConfiguration.socketOwner,
		SocketGroup:          createConfiguration.socketGroup,
		SocketPermissionMode: uint32(socketPermissionMode),
	})

	// Create the creation specification.
	specification := &forwardingsvc.CreationSpecification{
		Source:        source,
		Destination:   destination,
		Configuration: configuration,
		ConfigurationSource: &forwarding.Configuration{
			SocketOverwriteMode:  socketOverwriteModeSource,
			SocketOwner:          createConfiguration.socketOwnerSource,
			SocketGroup:          createConfiguration.socketGroupSource,
			SocketPermissionMode: uint32(socketPermissionModeSource),
		},
		ConfigurationDestination: &forwarding.Configuration{
			SocketOverwriteMode:  socketOverwriteModeDestination,
			SocketOwner:          createConfiguration.socketOwnerDestination,
			SocketGroup:          createConfiguration.socketGroupDestination,
			SocketPermissionMode: uint32(socketPermissionModeDestination),
		},
		Name:   createConfiguration.name,
		Labels: labels,
		Paused: createConfiguration.paused,
	}

	// Connect to the daemon and defer closure of the connection.
	daemonConnection, err := daemon.Connect(true, true)
	if err != nil {
		return fmt.Errorf("unable to connect to daemon: %w", err)
	}
	defer daemonConnection.Close()

	// Perform the create operation.
	identifier, err := CreateWithSpecification(daemonConnection, specification)
	if err != nil {
		return err
	}

	// Print the session identifier.
	fmt.Println("Created session", identifier)

	// Success.
	return nil
}

// createCommand is the create command.
var createCommand = &cobra.Command{
	Use:          "create <source> <destination>",
	Short:        "Create and start a new forwarding session",
	RunE:         createMain,
	SilenceUsage: true,
}

// createConfiguration stores configuration for the create command.
var createConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
	// name is the name specification for the session.
	name string
	// labels are the label specifications for the session.
	labels []string
	// paused indicates whether or not to create the session in a pre-paused
	// state.
	paused bool
	// noGlobalConfiguration specifies whether or not the global configuration
	// file should be ignored.
	noGlobalConfiguration bool
	// configurationFiles stores paths of additional files from which to load
	// default configuration.
	configurationFiles []string
	// socketOverwriteMode specifies the socket overwrite mode to use for the
	// session.
	socketOverwriteMode string
	// socketOverwriteModeSource specifies the socket overwrite mode to use for
	// the session, taking priority over socketOverwriteMode on source if
	// specified.
	socketOverwriteModeSource string
	// socketOverwriteModeDestination specifies the socket overwrite mode to use
	// for the session, taking priority over socketOverwriteMode on destination
	// if specified.
	socketOverwriteModeDestination string
	// socketOwner specifies the socket owner identifier to use new Unix domain
	// socket listeners, with endpoint-specific specifications taking priority.
	socketOwner string
	// socketOwnerSource specifies the socket owner identifier to use new Unix
	// domain socket listeners, taking priority over socketOwner on source if
	// specified.
	socketOwnerSource string
	// socketOwnerDestination specifies the socket owner identifier to use new
	// Unix domain socket listeners, taking priority over socketOwner on
	// destination if specified.
	socketOwnerDestination string
	// socketGroup specifies the socket owner identifier to use new Unix domain
	// socket listeners, with endpoint-specific specifications taking priority.
	socketGroup string
	// socketGroupSource specifies the socket owner identifier to use new Unix
	// domain socket listeners, taking priority over socketGroup on source if
	// specified.
	socketGroupSource string
	// socketGroupDestination specifies the socket owner identifier to use new
	// Unix domain socket listeners, taking priority over socketGroup on
	// destination if specified.
	socketGroupDestination string
	// socketPermissionMode specifies the socket permission mode to use for new
	// Unix domain socket listeners, with endpoint-specific specifications
	// taking priority.
	socketPermissionMode string
	// socketPermissionModeSource specifies the socket permission mode to use
	// for new Unix domain socket listeners on source, taking priority over
	// socketPermissionMode on source if specified.
	socketPermissionModeSource string
	// socketPermissionModeDestination specifies the socket permission mode to
	// use for new Unix domain socket listeners on destination, taking priority
	// over socketPermissionMode on destination if specified.
	socketPermissionModeDestination string
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
	flags.StringVarP(&createConfiguration.name, "name", "n", "", "Specify a name for the session")
	flags.StringSliceVarP(&createConfiguration.labels, "label", "l", nil, "Specify labels")

	// Wire up paused flags.
	flags.BoolVarP(&createConfiguration.paused, "paused", "p", false, "Create the session pre-paused")

	// Wire up general configuration flags.
	flags.BoolVar(&createConfiguration.noGlobalConfiguration, "no-global-configuration", false, "Ignore the global configuration file")
	flags.StringSliceVarP(&createConfiguration.configurationFiles, "configuration-file", "c", nil, "Specify additional files from which to load (and merge) default configuration parameters")

	// Wire up socket flags.
	flags.StringVar(&createConfiguration.socketOverwriteMode, "socket-overwrite-mode", "", "Specify socket overwrite mode (leave|overwrite)")
	flags.StringVar(&createConfiguration.socketOverwriteModeSource, "socket-overwrite-mode-source", "", "Specify socket overwrite mode for source (leave|overwrite)")
	flags.StringVar(&createConfiguration.socketOverwriteModeDestination, "socket-overwrite-mode-destination", "", "Specify socket overwrite mode for destination (leave|overwrite)")
	flags.StringVar(&createConfiguration.socketOwner, "socket-owner", "", "Specify socket owner")
	flags.StringVar(&createConfiguration.socketOwnerSource, "socket-owner-source", "", "Specify socket owner for source")
	flags.StringVar(&createConfiguration.socketOwnerDestination, "socket-owner-destination", "", "Specify socket owner for destination")
	flags.StringVar(&createConfiguration.socketGroup, "socket-group", "", "Specify socket group")
	flags.StringVar(&createConfiguration.socketGroupSource, "socket-group-source", "", "Specify socket group for source")
	flags.StringVar(&createConfiguration.socketGroupDestination, "socket-group-destination", "", "Specify socket group for destination")
	flags.StringVar(&createConfiguration.socketPermissionMode, "socket-permission-mode", "", "Specify socket permission mode")
	flags.StringVar(&createConfiguration.socketPermissionModeSource, "socket-permission-mode-source", "", "Specify socket permission mode for source")
	flags.StringVar(&createConfiguration.socketPermissionModeDestination, "socket-permission-mode-destination", "", "Specify socket permission mode for destination")
}
