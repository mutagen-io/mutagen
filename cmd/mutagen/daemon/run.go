package daemon

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/daemon"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/identifier"
	"github.com/mutagen-io/mutagen/pkg/ipc"
	"github.com/mutagen-io/mutagen/pkg/logging"
	daemonsvc "github.com/mutagen-io/mutagen/pkg/service/daemon"
	forwardingsvc "github.com/mutagen-io/mutagen/pkg/service/forwarding"
	promptingsvc "github.com/mutagen-io/mutagen/pkg/service/prompting"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"

	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/docker"
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/local"
	_ "github.com/mutagen-io/mutagen/pkg/forwarding/protocols/ssh"
	_ "github.com/mutagen-io/mutagen/pkg/synchronization/protocols/docker"
	_ "github.com/mutagen-io/mutagen/pkg/synchronization/protocols/local"
	_ "github.com/mutagen-io/mutagen/pkg/synchronization/protocols/ssh"
)

// runMain is the entry point for the run command.
func runMain(_ *cobra.Command, _ []string) error {
	// Attempt to acquire the daemon lock and defer its release.
	lock, err := daemon.AcquireLock()
	if err != nil {
		return fmt.Errorf("unable to acquire daemon lock: %w", err)
	}
	defer lock.Release()

	// Create a channel to track termination signals. We do this before creating
	// and starting other infrastructure so that we can ensure things terminate
	// smoothly, not mid-initialization.
	signalTermination := make(chan os.Signal, 1)
	signal.Notify(signalTermination, cmd.TerminationSignals...)

	// Open the daemon log and defer its closure.
	logFile, err := daemon.OpenLog()
	if err != nil {
		return fmt.Errorf("unable to open daemon log: %w", err)
	}
	defer logFile.Close()

	// Create the root logger.
	logger := logging.NewLogger(io.MultiWriter(logFile, os.Stderr))

	// Create a forwarding session manager and defer its shutdown.
	forwardingManager, err := forwarding.NewManager(logger.Sublogger("forwarding"))
	if err != nil {
		return fmt.Errorf("unable to create forwarding session manager: %w", err)
	}
	defer forwardingManager.Shutdown()

	// Create a synchronization session manager and defer its shutdown.
	synchronizationManager, err := synchronization.NewManager(logger.Sublogger("synchronization"))
	if err != nil {
		return fmt.Errorf("unable to create synchronization session manager: %w", err)
	}
	defer synchronizationManager.Shutdown()

	// Create the gRPC server and defer its stoppage. We use a hard stop rather
	// than a graceful stop so that it doesn't hang on open requests.
	server := grpc.NewServer(
		grpc.MaxSendMsgSize(grpcutil.MaximumMessageSize),
		grpc.MaxRecvMsgSize(grpcutil.MaximumMessageSize),
	)
	defer server.Stop()

	// Create the daemon server, defer its shutdown, and register it.
	daemonServer := daemonsvc.NewServer()
	defer daemonServer.Shutdown()
	daemonsvc.RegisterDaemonServer(server, daemonServer)

	// Create and register the prompt server.
	promptingsvc.RegisterPromptingServer(server, promptingsvc.NewServer())

	// Create and register the forwarding server.
	forwardingServer := forwardingsvc.NewServer(forwardingManager)
	forwardingsvc.RegisterForwardingServer(server, forwardingServer)

	// Create and register the synchronization server.
	synchronizationServer := synchronizationsvc.NewServer(synchronizationManager)
	synchronizationsvc.RegisterSynchronizationServer(server, synchronizationServer)

	// Compute the path to the daemon IPC endpoint.
	endpoint, err := daemon.EndpointPath()
	if err != nil {
		return fmt.Errorf("unable to compute IPC endpoint path: %w", err)
	}

	// Create the daemon listener and defer its closure. Since we hold the
	// daemon lock, we preemptively remove any existing socket since it (should)
	// be stale.
	os.Remove(endpoint)
	ipcListener, err := ipc.NewListener(endpoint)
	if err != nil {
		return fmt.Errorf("unable to create IPC listener: %w", err)
	}
	defer ipcListener.Close()

	// Serve incoming connections in a separate Goroutine, watching for serving
	// failure.
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(ipcListener)
	}()

	// Compute the daemon token storage path.
	tokenPath, err := daemon.TokenPath()
	if err != nil {
		return fmt.Errorf("unable to compute token path: %w", err)
	}

	// Load the daemon token. If it's missing or invalid, then generate a new
	// one and write it to disk.
	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			tokenBytes = nil
		} else {
			return fmt.Errorf("unable to load token: %w", err)
		}
	} else if !utf8.Valid(tokenBytes) {
		tokenBytes = nil
	}
	token := string(tokenBytes)
	if !identifier.IsValid(token, false) {
		token, err = identifier.New(identifier.PrefixToken)
		if err != nil {
			return fmt.Errorf("unable to generate token: %w", err)
		}
		if err := filesystem.WriteFileAtomic(tokenPath, []byte(token), 0600); err != nil {
			return fmt.Errorf("unable to write daemon token to disk: %w", err)
		}
	}

	// Compute the daemon TCP port.
	var port uint16
	if p, ok := os.LookupEnv("MUTAGEN_DAEMON_TCP_PORT"); ok {
		if p16, err := strconv.ParseUint(p, 10, 16); err != nil {
			return fmt.Errorf("invalid port (%s) specified in environment", p)
		} else {
			port = uint16(p16)
		}
	} else {
		port = daemon.DefaultPort
	}

	// Create the daemon TCP listener and defer its closure.
	bind := fmt.Sprintf("%s:%d", daemon.Host, port)
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		return fmt.Errorf("unable to bind to daemon TCP port: %w", err)
	}
	defer listener.Close()

	// If using a dynamic port, determine which port was allocated.
	if port == 0 {
		address, ok := listener.Addr().(*net.TCPAddr)
		if !ok {
			return errors.New("invalid listener address type")
		}
		port = uint16(address.Port)
	}

	// Write the daemon TCP port to disk and defer its removal.
	portPath, err := daemon.PortPath()
	if err != nil {
		return fmt.Errorf("unable to compute daemon port path: %w", err)
	}
	portBytes := []byte(fmt.Sprintf("%d", port))
	if err := filesystem.WriteFileAtomic(portPath, portBytes, 0600); err != nil {
		return fmt.Errorf("unable to write daemon port to disk: %w", err)
	}
	defer os.Remove(portPath)

	// Wait for termination from a signal, the daemon service, or the gRPC
	// server. We treat termination via the daemon service as a non-error.
	select {
	case sig := <-signalTermination:
		logger.Info("Received termination signal:", sig)
		return nil
	case <-daemonServer.Termination:
		logger.Info("Received termination request")
		return nil
	case err = <-serverErrors:
		logger.Error("Daemon API server terminated abnormally:", err)
		return fmt.Errorf("daemon API server terminated abnormally: %w", err)
	}
}

// runCommand is the run command.
var runCommand = &cobra.Command{
	Use:          "run",
	Short:        "Run the Mutagen daemon",
	Args:         cmd.DisallowArguments,
	Hidden:       true,
	RunE:         runMain,
	SilenceUsage: true,
}

// runConfiguration stores configuration for the run command.
var runConfiguration struct {
	// help indicates whether or not to show help information and exit.
	help bool
}

func init() {
	// Grab a handle for the command line flags.
	flags := runCommand.Flags()

	// Disable alphabetical sorting of flags in help output.
	flags.SortFlags = false

	// Manually add a help flag to override the default message. Cobra will
	// still implement its logic automatically.
	flags.BoolVarP(&runConfiguration.help, "help", "h", false, "Show help information")
}
