package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"

	"github.com/julienschmidt/httprouter"

	"github.com/mutagen-io/mutagen/cmd"

	"github.com/mutagen-io/mutagen/pkg/api"
	"github.com/mutagen-io/mutagen/pkg/daemon"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/grpcutil"
	"github.com/mutagen-io/mutagen/pkg/housekeeping"
	"github.com/mutagen-io/mutagen/pkg/identifier"
	"github.com/mutagen-io/mutagen/pkg/ipc"
	"github.com/mutagen-io/mutagen/pkg/logging"
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
	terminationSignals := make(chan os.Signal, 1)
	signal.Notify(terminationSignals, cmd.TerminationSignals...)

	// Open the daemon log and defer its closure.
	logFile, err := daemon.OpenLog()
	if err != nil {
		return fmt.Errorf("unable to open daemon log: %w", err)
	}
	defer logFile.Close()

	// Create the root logger.
	logger := logging.NewLogger(io.MultiWriter(logFile, os.Stderr))

	// Set up regular housekeeping and defer its shutdown.
	housekeepingCtx, cancelHousekeeping := context.WithCancel(context.Background())
	defer cancelHousekeeping()
	go housekeeping.HousekeepRegularly(housekeepingCtx, logger.Sublogger("housekeeping"))

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
	grpcServer := grpc.NewServer(
		grpc.MaxSendMsgSize(grpcutil.MaximumMessageSize),
		grpc.MaxRecvMsgSize(grpcutil.MaximumMessageSize),
	)
	defer grpcServer.Stop()

	// Create and register the prompt server.
	promptingsvc.RegisterPromptingServer(grpcServer, promptingsvc.NewServer())

	// Create and register the forwarding server.
	forwardingServer := forwardingsvc.NewServer(forwardingManager)
	forwardingsvc.RegisterForwardingServer(grpcServer, forwardingServer)

	// Create and register the synchronization server.
	synchronizationServer := synchronizationsvc.NewServer(synchronizationManager)
	synchronizationsvc.RegisterSynchronizationServer(grpcServer, synchronizationServer)

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

	// Serve incoming connections in a separate Goroutine and watch for failure.
	grpcServerErrors := make(chan error, 1)
	go func() {
		grpcServerErrors <- grpcServer.Serve(ipcListener)
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

	// Create the HTTP request router.
	router := httprouter.New()

	// Disable automatic trailing slash redirection in the router.
	router.RedirectTrailingSlash = false

	// Disable automated path fixing in the router.
	router.RedirectFixedPath = false

	// Prevent supported method information disclosure when an incorrect method
	// is used on a path supporting other methods.
	router.HandleMethodNotAllowed = false

	// Prevent supported method information disclosure via OPTIONS requests.
	router.HandleOPTIONS = false

	// Create the daemon service and register its endpoints.
	daemonService := daemon.NewService()
	daemonService.Register(router)

	// TODO: Create the agent service and register its endpoints.

	// TODO: Create the prompting service and register its endpoints.

	// TODO: Create the synchronization service and register its endpoints.

	// TODO: Create the forwarding service and register its endpoints.

	// Abstract the router to a generic handler so that we can apply middleware.
	handler := http.Handler(router)

	// Require authentication.
	handler = api.RequireAuthentication(handler, token)

	// Add response security headers.
	handler = api.AddSecurityHeaders(handler)

	// TODO: Set up request logging for debugging or tracing.

	// Create the daemon HTTP server. We intentionally avoid setting a write
	// timeout because some API requests use indefinite polling.
	// TODO: Redirect error logging.
	server := &http.Server{
		Handler:     handler,
		ReadTimeout: api.ReadTimeout,
		IdleTimeout: api.IdleTimeout,
	}

	// Serve incoming connections in a separate Goroutine and watch for failure.
	// We defer a hard shutdown of the server because we don't want blocking
	// requests to block daemon shutdown.
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Serve(listener)
	}()
	defer server.Close()

	// Wait for a termination signal, a termination request via the daemon
	// service, or an error from the API server.
	select {
	case s := <-terminationSignals:
		logger.Info("Received termination signal:", s)
		return nil
	case <-daemonService.Done():
		logger.Info("Received termination request")
		return nil
	case err = <-serverErrors:
		logger.Error("Daemon server terminated abnormally:", err)
		return fmt.Errorf("daemon server terminated abnormally: %w", err)
	case err = <-grpcServerErrors:
		logger.Error("Daemon gRPC server terminated abnormally:", err)
		return fmt.Errorf("daemon gRPC server terminated abnormally: %w", err)
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
