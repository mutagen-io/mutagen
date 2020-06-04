package agent

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/process"
	"github.com/mutagen-io/mutagen/pkg/prompting"
	"github.com/mutagen-io/mutagen/pkg/stream"
)

const (
	// agentKillDelay is the maximum amount of time that Mutagen will wait for
	// an agent process to exit on its own in the event of an error before
	// issuing a kill request. It's necessary to make this non-zero because
	// agent transport executables need to be allowed time to exit on failure so
	// that we can get their true exit codes instead of killing them immediately
	// on a handshake error.
	agentKillDelay = 5 * time.Second
)

// connect connects to an agent-based endpoint using the specified transport,
// connection mode, and prompter. It accepts a hint as to whether or not the
// remote environment is cmd.exe-based and returns hints as to whether or not
// installation should be attempted and whether or not the remote environment is
// cmd.exe-based.
func connect(logger *logging.Logger, transport Transport, mode, prompter string, cmdExe bool) (net.Conn, bool, bool, error) {
	// Compute the agent invocation command, relative to the user's home
	// directory on the remote. Unless we have reason to assume that this is a
	// cmd.exe environment, we construct a path using forward slashes. This will
	// work for all POSIX systems and POSIX-like environments on Windows. If we
	// know we're hitting a cmd.exe environment, then we use backslashes,
	// otherwise the invocation won't work. Watching for cmd.exe to fail on
	// commands with forward slashes is actually the way that we detect cmd.exe
	// environments.
	//
	// HACK: We're assuming that none of these path components have spaces in
	// them, but since we control all of them, this is probably okay.
	//
	// HACK: When invoking on Windows systems (whether inside a POSIX
	// environment or cmd.exe), we can leave the "exe" suffix off the target
	// name. Fortunately this allows us to also avoid having to try the
	// combination of forward slashes + ".exe" for Windows POSIX environments.
	pathSeparator := "/"
	if cmdExe {
		pathSeparator = "\\"
	}
	dataDirectoryName := filesystem.MutagenDataDirectoryName
	if mutagen.DevelopmentModeEnabled {
		dataDirectoryName = filesystem.MutagenDataDirectoryDevelopmentName
	}
	agentInvocationPath := strings.Join([]string{
		dataDirectoryName,
		filesystem.MutagenAgentsDirectoryName,
		mutagen.Version,
		BaseName,
	}, pathSeparator)

	// Compute the command to invoke.
	command := fmt.Sprintf("%s %s", agentInvocationPath, mode)

	// Create an agent process.
	message := "Connecting to agent (POSIX)..."
	if cmdExe {
		message = "Connecting to agent (Windows)..."
	}
	if err := prompting.Message(prompter, message); err != nil {
		return nil, false, false, errors.Wrap(err, "unable to message prompter")
	}
	agentProcess, err := transport.Command(command)
	if err != nil {
		return nil, false, false, errors.Wrap(err, "unable to create agent command")
	}

	// Create a connection that wraps the process' standard input/output. We
	// set a non-zero kill delay so that, if there's a handshake failure, the
	// process will be allowed to exit with its natural exit code (instead of an
	// exit code due to forced termination) that we can use to diagnose the
	// connection issue.
	connection, err := process.NewConnection(agentProcess, agentKillDelay)
	if err != nil {
		return nil, false, false, errors.Wrap(err, "unable to create agent process connection")
	}

	// Create a buffer that we can use to capture the process' standard error
	// output in order to give better feedback when there's an error.
	errorBuffer := bytes.NewBuffer(nil)

	// Wrap the error buffer in a valveWriter and defer the closure of that
	// writer. This avoids continuing to pipe output into the buffer for the
	// lifetime of the process.
	errorWriter := stream.NewValveWriter(errorBuffer)
	defer errorWriter.Shut()

	// Redirect the process' standard error output to a tee'd writer that writes
	// to both our buffer (via the valveWriter) and the logger.
	agentProcess.Stderr = io.MultiWriter(errorWriter, logger.Sublogger("remote").Writer(logging.LevelInfo))

	// Start the process.
	if err = agentProcess.Start(); err != nil {
		return nil, false, false, errors.Wrap(err, "unable to start agent process")
	}

	// Perform a handshake with the remote to ensure that we're talking with a
	// Mutagen agent.
	if err := ClientHandshake(connection); err != nil {
		// Close the connection to ensure that the underlying process and its
		// I/O-forwarding Goroutines have terminated. The error returned from
		// Close will be non-nil if the process exits with a non-0 exit code, so
		// we don't want to check it, but process.Connection guarantees that if
		// Close returns, the underlying process has fully terminated, which is
		// all we care about.
		connection.Close()

		// Extract error output and ensure it's UTF-8.
		errorOutput := errorBuffer.String()
		if !utf8.ValidString(errorOutput) {
			return nil, false, false, errors.New("remote did not return UTF-8 output")
		}

		// See if we can understand the exact nature of the failure. In
		// particular, we want to identify whether or not we should try to
		// (re-)install the agent binary and whether or not we're talking to a
		// Windows cmd.exe environment. We have to delegate this responsibility
		// to the transport, because each has different error classification
		// mechanisms. If the transport can't figure it out but we have some
		// error output, then give it to the user, because they're probably in a
		// better place to interpret the error output then they are to interpret
		// the transport's reason for classification failure. If we don't have
		// error output, then just tell the user why the transport failed to
		// classify the failure.
		tryInstall, cmdExe, err := transport.ClassifyError(agentProcess.ProcessState, errorOutput)
		if err != nil {
			if errorOutput != "" {
				return nil, false, false, errors.Errorf(
					"agent handshake failed with error output:\n%s",
					strings.TrimSpace(errorOutput),
				)
			}
			return nil, false, false, errors.Wrap(err, "unable to classify agent handshake error")
		}

		// The transport was able to classify the error, so return that
		// information.
		return nil, tryInstall, cmdExe, errors.New("unable to handshake with agent process")
	}

	// Now that we've successfully connected, disable the kill delay on the
	// process connection.
	connection.SetKillDelay(time.Duration(0))

	// Perform a version handshake.
	if err := mutagen.ClientVersionHandshake(connection); err != nil {
		connection.Close()
		return nil, false, false, errors.Wrap(err, "version handshake error")
	}

	// Done.
	return connection, false, false, nil
}

// Dial connects to an agent-based endpoint using the specified transport,
// connection mode, and prompter.
func Dial(logger *logging.Logger, transport Transport, mode, prompter string) (net.Conn, error) {
	// Validate that the mode is sane.
	if !(mode == ModeSynchronizer || mode == ModeForwarder) {
		panic("invalid agent dial mode")
	}

	// Attempt a connection. If this fails but we detect a Windows cmd.exe
	// environment in the process, then re-attempt a connection under the
	// cmd.exe assumption.
	connection, tryInstall, cmdExe, err := connect(logger, transport, mode, prompter, false)
	if err == nil {
		return connection, nil
	} else if cmdExe {
		connection, tryInstall, cmdExe, err = connect(logger, transport, mode, prompter, true)
		if err == nil {
			return connection, nil
		}
	}

	// If connection attempts have failed, then check whether or not an install
	// is recommended. If not, then bail.
	if !tryInstall {
		return nil, err
	}

	// Attempt to install.
	if err := install(logger, transport, prompter); err != nil {
		return nil, errors.Wrap(err, "unable to install agent")
	}

	// Re-attempt connectivity.
	connection, _, _, err = connect(logger, transport, mode, prompter, cmdExe)
	if err != nil {
		return nil, err
	}
	return connection, nil
}
