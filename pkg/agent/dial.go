package agent

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	transportpkg "github.com/mutagen-io/mutagen/pkg/agent/transport"
	"github.com/mutagen-io/mutagen/pkg/filesystem"
	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/prompting"
	streampkg "github.com/mutagen-io/mutagen/pkg/stream"
)

const (
	// agentTerminationDelay is the maximum amount of time that Mutagen will
	// wait for an agent process to terminate on its own in the event of a
	// handshake error before forcing termination.
	agentTerminationDelay = 5 * time.Second
	// agentErrorInMemoryCutoff is the maximum number of bytes that Mutagen will
	// capture in memory from the standard error output of an agent process.
	agentErrorInMemoryCutoff = 32 * 1024
)

// connect connects to an agent-based endpoint using the specified transport,
// connection mode, and prompter. It accepts a hint as to whether or not the
// remote environment is cmd.exe-based and returns hints as to whether or not
// installation should be attempted and whether or not the remote environment is
// cmd.exe-based.
func connect(logger *logging.Logger, transport Transport, mode, prompter string, cmdExe bool) (io.ReadWriteCloser, bool, bool, error) {
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
	command := fmt.Sprintf("%s %s --%s=%s", agentInvocationPath, mode, FlagLogLevel, logger.Level())

	// Set up (but do not start) an agent process.
	message := "Connecting to agent (POSIX)..."
	if cmdExe {
		message = "Connecting to agent (Windows)..."
	}
	if err := prompting.Message(prompter, message); err != nil {
		return nil, false, false, fmt.Errorf("unable to message prompter: %w", err)
	}
	agentProcess, err := transport.Command(command)
	if err != nil {
		return nil, false, false, fmt.Errorf("unable to create agent command: %w", err)
	}

	// Create a buffer that we can use to capture the process' standard error
	// output in order to give better feedback when there's an error.
	errorBuffer := bytes.NewBuffer(nil)

	// Create a cutoff for the error buffer that avoids using large amounts of
	// memory (while still being sufficiently large to capture any reasonable
	// human-readable error message).
	errorCutoff := streampkg.NewCutoffWriter(errorBuffer, agentErrorInMemoryCutoff)

	// Create a valve that we can use to stop recording the error output once
	// this function returns (at which point the error will already have been
	// captured or not have occurred).
	errorValve := streampkg.NewValveWriter(errorCutoff)
	defer errorValve.Shut()

	// Create a splitter that will forward standard error output to both the
	// error buffer and the logger. The error log level we apply here only
	// applies to non-log messages printed to standard error - all log messages
	// routed through standard error have their levels forwarded.
	errorTee := io.MultiWriter(errorValve, logger.Writer(logging.LevelError))

	// Create a transport stream to communicate with the process and forward
	// standard error output. Set a non-zero termination delay for the stream so
	// that (in the event of a handshake failure) the process will be allowed to
	// exit with its natural exit code (instead of an exit code due to forced
	// termination) and will be able to yield some error output for diagnosing
	// the issue.
	stream, err := transportpkg.NewStream(agentProcess, errorTee)
	if err != nil {
		return nil, false, false, fmt.Errorf("unable to create agent process stream: %w", err)
	}
	stream.SetTerminationDelay(agentTerminationDelay)

	// Start the process.
	if err = agentProcess.Start(); err != nil {
		return nil, false, false, fmt.Errorf("unable to start agent process: %w", err)
	}

	// Perform a handshake with the remote to ensure that we're talking with a
	// Mutagen agent.
	if err := ClientHandshake(stream); err != nil {
		// Close the stream to ensure that the underlying process and any
		// I/O-forwarding Goroutines have terminated. The error returned from
		// Close will be non-nil if the process exits with a non-0 exit code, so
		// we don't want to check it, but transport.Stream guarantees that if
		// Close returns, then the underlying process has fully terminated,
		// which is all we care about.
		stream.Close()

		// Extract any error output, ensure that it's UTF-8, and strip out any
		// whitespace (primarily trailing newlines).
		errorOutput := errorBuffer.String()
		if !utf8.ValidString(errorOutput) {
			return nil, false, false, errors.New("remote did not return UTF-8 output")
		}
		errorOutput = strings.TrimSpace(errorOutput)

		// Wrap up the handshake error with additional context.
		if errorOutput != "" {
			err = fmt.Errorf("unable to handshake with agent process: %w (error output: %s)", err, errorOutput)
		} else {
			err = fmt.Errorf("unable to handshake with agent process: %w", err)
		}

		// See if we can classify the exact nature of the handshake failure. In
		// particular, we want to identify whether or not we should try to
		// (re-)install the agent binary and whether or not we're talking to a
		// Windows cmd.exe environment. We have to delegate this responsibility
		// to the transport, because each transport has different error
		// classification mechanisms. We don't bother returning classification
		// failure errors because they don't contain any useful information; the
		// user is far better off trying to interpret the original error and
		// error output from the handshake failure.
		tryInstall, cmdExe, classifyErr := transport.ClassifyError(agentProcess.ProcessState, errorOutput)
		if classifyErr != nil {
			return nil, false, false, err
		}
		return nil, tryInstall, cmdExe, err
	}

	// Now that we've successfully connected, disable the termination delay on
	// the process stream.
	stream.SetTerminationDelay(time.Duration(0))

	// Perform a version handshake.
	if err := mutagen.ClientVersionHandshake(stream); err != nil {
		stream.Close()
		return nil, false, false, fmt.Errorf("version handshake error: %w", err)
	}

	// Done.
	return stream, false, false, nil
}

// Dial connects to an agent-based endpoint using the specified transport,
// connection mode, and prompter.
func Dial(logger *logging.Logger, transport Transport, mode, prompter string) (io.ReadWriteCloser, error) {
	// Validate that the mode is sane.
	if !(mode == CommandSynchronizer || mode == CommandForwarder) {
		return nil, errors.New("invalid agent dial mode")
	}

	// Attempt a connection. If this fails but we detect a Windows cmd.exe
	// environment in the process, then re-attempt a connection under the
	// cmd.exe assumption.
	stream, tryInstall, cmdExe, err := connect(logger, transport, mode, prompter, false)
	if err == nil {
		return stream, nil
	} else if cmdExe {
		stream, tryInstall, cmdExe, err = connect(logger, transport, mode, prompter, true)
		if err == nil {
			return stream, nil
		}
	}

	// If connection attempts have failed, then check whether or not an install
	// is recommended. If not, then bail.
	if !tryInstall {
		return nil, err
	}

	// Attempt to install.
	if err := install(logger, transport, prompter); err != nil {
		return nil, fmt.Errorf("unable to install agent: %w", err)
	}

	// Re-attempt connectivity.
	stream, _, _, err = connect(logger, transport, mode, prompter, cmdExe)
	if err != nil {
		return nil, err
	}
	return stream, nil
}
