package agent

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/process"
	"github.com/havoc-io/mutagen/pkg/prompt"
	"github.com/havoc-io/mutagen/pkg/remote"
	"github.com/havoc-io/mutagen/pkg/session"
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

func connect(
	transport Transport,
	prompter string,
	cmdExe bool,
	root,
	session string,
	version session.Version,
	configuration *session.Configuration,
	alpha bool,
) (session.Endpoint, bool, bool, error) {
	// Compute the agent invocation command, relative to the user's home
	// directory on the remote. Unless we have reason to assume that this is a
	// cmd.exe environment, we construct a path using forward slashes. This will
	// work for all POSIX systems and POSIX-like environments on Windows. If we
	// know we're hitting a cmd.exe environment, then we use backslashes,
	// otherwise the invocation won't work. Watching for cmd.exe to fail on
	// commands with forward slashes is actually the way that we detect cmd.exe
	// environments.
	// HACK: We're assuming that none of these path components have spaces in
	// them, but since we control all of them, this is probably okay.
	// HACK: When invoking on Windows systems (whether inside a POSIX
	// environment or cmd.exe), we can leave the "exe" suffix off the target
	// name. Fortunately this allows us to also avoid having to try the
	// combination of forward slashes + ".exe" for Windows POSIX environments.
	pathSeparator := "/"
	if cmdExe {
		pathSeparator = "\\"
	}
	agentInvocationPath := strings.Join([]string{
		filesystem.MutagenDirectoryName,
		agentsDirectoryName,
		mutagen.Version,
		agentBaseName,
	}, pathSeparator)

	// Compute the command to invoke.
	// FIXME: This shouldn't be hardcoded to /root
	command := fmt.Sprintf("/root/%s %s", agentInvocationPath, ModeEndpoint)

	// Create an agent process.
	message := "Connecting to agent (POSIX)..."
	if cmdExe {
		message = "Connecting to agent (Windows)..."
	}
	if err := prompt.Message(prompter, message); err != nil {
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

	// Redirect the process' standard error output to a buffer so that we can
	// give better feedback in errors. This might be a bit dangerous since this
	// buffer will be attached for the lifetime of the process and we don't know
	// exactly how much output will be received (and thus we could buffer a
	// large amount of it in memory), but generally speaking our transport
	// commands don't spit out too much error output, and the agent doesn't spit
	// out any.
	// TODO: If we do start seeing large allocations in these buffers, a simple
	// size-limited buffer might suffice, at least to get some of the error
	// message.
	// TODO: Since this problem will likely be shared with custom protocols
	// (which will invoke transport executables), it would be good to implement
	// a shared solution.
	errorBuffer := bytes.NewBuffer(nil)
	agentProcess.Stderr = errorBuffer

	// Start the process.
	if err = agentProcess.Start(); err != nil {
		return nil, false, false, errors.Wrap(err, "unable to start agent process")
	}

	// Wrap the connection in an endpoint client and handle errors that may have
	// arisen during the handshake process. Specifically, we look for transport
	// errors that occur during handshake, because that's an indication that our
	// agent transport process is not functioning correctly. If that's the case,
	// we wait for the agent transport process to exit (which we know it will
	// because the NewEndpointClient method will close the connection (hence
	// terminating the process) on failure), and probe the issue.
	endpoint, err := remote.NewEndpointClient(connection, root, session, version, configuration, alpha)
	if remote.IsHandshakeTransportError(err) {
		// At this point, we know that the agent process and its I/O forwarding
		// Goroutines have terminated because NewEndpointClient will have closed
		// the connection on error and the Close method won't return until the
		// process has fully terminated. As a result, it's safe to touch the
		// error buffer and process state for the agent process at this point.

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
	} else if err != nil {
		return nil, false, false, errors.Wrap(err, "unable to create endpoint client")
	}

	// Now that we've successfully connected, disable the kill delay on the
	// process connection.
	connection.SetKillDelay(time.Duration(0))

	// Done.
	return endpoint, false, false, nil
}

// Dial connects to an agent-based endpoint using the specified transport,
// prompter, and endpoint metadata.
func Dial(
	transport Transport,
	prompter,
	root,
	session string,
	version session.Version,
	configuration *session.Configuration,
	alpha bool,
) (session.Endpoint, error) {
	// Attempt a connection. If this fails but we detect a Windows cmd.exe
	// environment in the process, then re-attempt a connection under the
	// cmd.exe assumption.
	endpoint, tryInstall, cmdExe, err :=
		connect(transport, prompter, false, root, session, version, configuration, alpha)
	if err == nil {
		return endpoint, nil
	} else if cmdExe {
		endpoint, tryInstall, cmdExe, err =
			connect(transport, prompter, true, root, session, version, configuration, alpha)
		if err == nil {
			return endpoint, nil
		}
	}

	// If connection attempts have failed, then check whether or not an install
	// is recommended. If not, then bail.
	if !tryInstall {
		return nil, err
	}

	// Attempt to install.
	if err := install(transport, prompter); err != nil {
		return nil, errors.Wrap(err, "unable to install agent")
	}

	// Re-attempt connectivity.
	endpoint, _, _, err = connect(transport, prompter, cmdExe, root, session, version, configuration, alpha)
	if err != nil {
		return nil, err
	}
	return endpoint, nil
}
