package agent

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/process"
	"github.com/havoc-io/mutagen/pkg/prompt"
	"github.com/havoc-io/mutagen/pkg/remote"
	"github.com/havoc-io/mutagen/pkg/session"
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
	command := fmt.Sprintf("%s %s", agentInvocationPath, ModeEndpoint)

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

	// Create a connection that wrap's the process' standard input/output.
	connection, err := process.NewConnection(agentProcess)
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
		// Wait for the process to complete. We need to do this before touching
		// the error buffer because it isn't safe for concurrent usage, and
		// until Wait completes, the I/O forwarding Goroutines can still be
		// running.
		processErr := agentProcess.Wait()

		// Extract error output and ensure it's UTF-8.
		errorOutput := errorBuffer.String()
		if !utf8.ValidString(errorOutput) {
			return nil, false, false, errors.New("remote did not return UTF-8 output")
		}

		// See if we can understand the exact nature of the failure. In
		// particular, we want to identify whether or not we should try to
		// (re-)install the agent binary and whether or not we're talking to a
		// Windows cmd.exe environment. The logic here is a little bit tricky
		// and fragile, because the way that our underlying transports return
		// error codes and error output information is all over the map.
		//
		// For example, if SSH is successful in connecting to a remote, it
		// always forwards the exit code from the remote shell as its own exit
		// code. Docker on the other hand seems to have its own pseudo-POSIX
		// internal codes (see https://github.com/moby/moby/pull/14012) which
		// don't correspond exactly to what's returned from the remote platform.
		// For example, what would be a POSIX shell "command not found" (127)
		// error would be forwarded directly by SSH, but aliased to a POSIX
		// "invalid command" (126) error by Docker. It seems that Docker uses
		// these POSIX-y exit codes even when talking to a Windows container.
		//
		// SSH also returns standard error output on standard error, while
		// Docker merges it with the standard output stream, and it is thus
		// gobbled up by the endpoint client (which fortunately rejects it using
		// a magic number scheme).
		//
		// Suffice it to say, the problem is complex. We may eventually need to
		// add an error classification method to the Transport interface to aid
		// in determining this information. In the case of Docker, we might need
		// to tee the standard output stream into both the endpoint client and
		// a limited-size buffer to help diagnose failures.
		if process.IsPOSIXShellInvalidCommand(processErr) {
			// TODO: This code returned from Docker doesn't necessarily indicate
			// that the container is a POSIX environment - it may be a Windows
			// container, because Docker still uses the 126 exit code even in
			// that case. We need to figure out how to identify Windows
			// containers using Docker. Fortunately, we already probe this in
			// the Docker transport, so we can probably expose an "is windows"
			// hint via the Transport interface.
			return nil, true, false, errors.New("invalid command")
		} else if process.IsPOSIXShellCommandNotFound(processErr) {
			return nil, true, false, errors.New("command not found")
		} else if process.OutputIsWindowsInvalidCommand(errorOutput) {
			// A Windows invalid command error doesn't necessarily indicate that
			// the agent isn't installed, but merely that we were using a POSIX
			// shell syntax in a Windows cmd.exe environment. In this case, we
			// should try to connect again under the cmd.exe hypothesis before
			// attempting an install. If we conect under the cmd.exe hypothesis
			// and the agent really isn't there, then we'll see a Windows
			// "command not found" error next, and then we can try to install
			// the agent.
			// HACK: Returning false for "try install" here is using some
			// knowledge of the caller's behavior, but I think that's okay since
			// they're so tightly coupled anyway.
			return nil, false, true, errors.New("invalid command")
		} else if process.OutputIsWindowsCommandNotFound(errorOutput) {
			return nil, true, true, errors.New("command not found")
		}

		// Otherwise, check if there is any error output that might illuminate
		// what happened. We let this overrule any err value here since that
		// value will probably just be an EOF.
		if errorOutput != "" {
			return nil, false, false, errors.Errorf(
				"agent process failed with error output:\n%s",
				strings.TrimSpace(errorOutput),
			)
		}

		// Otherwise just wrap up whatever error we have.
		return nil, false, false, errors.Wrap(err, "unable to handshake with agent process")
	} else if err != nil {
		return nil, false, false, errors.Wrap(err, "unable to create endpoint client")
	}

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
