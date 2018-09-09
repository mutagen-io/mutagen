package agent

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/google/uuid"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/process"
	"github.com/havoc-io/mutagen/pkg/prompt"
	"github.com/havoc-io/mutagen/pkg/remote"
	"github.com/havoc-io/mutagen/pkg/session"
)

func probePOSIX(transport Transport) (string, string, error) {
	// Try to invoke uname and print kernel and machine name.
	unameSMBytes, err := output(transport, "uname -s -m")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to invoke uname")
	} else if !utf8.Valid(unameSMBytes) {
		return "", "", errors.New("remote output is not UTF-8 encoded")
	}

	// Parse uname output.
	unameSM := strings.Split(strings.TrimSpace(string(unameSMBytes)), " ")
	if len(unameSM) != 2 {
		return "", "", errors.New("invalid uname output")
	}
	unameS := unameSM[0]
	unameM := unameSM[1]

	// Translate GOOS.
	var goos string
	if unameSIsWindowsPosix(unameS) {
		goos = "windows"
	} else if g, ok := unameSToGOOS[unameS]; ok {
		goos = g
	} else {
		return "", "", errors.New("unknown platform")
	}

	// Translate GOARCH.
	goarch, ok := unameMToGOARCH[unameM]
	if !ok {
		return "", "", errors.New("unknown architecture")
	}

	// Success.
	return goos, goarch, nil
}

func probeWindows(transport Transport) (string, string, error) {
	// Attempt to dump the remote environment.
	outputBytes, err := output(transport, "cmd /c set")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to invoke remote environment printing")
	} else if !utf8.Valid(outputBytes) {
		return "", "", errors.New("remote output is not UTF-8 encoded")
	}

	// Parse the environment output.
	output := string(outputBytes)
	output = strings.Replace(output, "\r\n", "\n", -1)
	output = strings.TrimSpace(output)
	environment := strings.Split(output, "\n")

	// Extract the OS and PROCESSOR_ARCHITECTURE environment variables.
	var os, processorArchitecture string
	for _, e := range environment {
		if strings.HasPrefix(e, "OS=") {
			os = e[3:]
		} else if strings.HasPrefix(e, "PROCESSOR_ARCHITECTURE=") {
			processorArchitecture = e[23:]
		}
	}

	// Translate to GOOS.
	goos, ok := osEnvToGOOS[os]
	if !ok {
		return "", "", errors.New("unknown platform")
	}

	// Translate to GOARCH.
	goarch, ok := processorArchitectureEnvToGOARCH[processorArchitecture]
	if !ok {
		return "", "", errors.New("unknown architecture")
	}

	// Success.
	return goos, goarch, nil
}

// probe attempts to identify the properties of the target platform, namely
// GOOS, GOARCH, and whether or not it's a POSIX environment (which it might be
// even on Windows).
func probe(transport Transport, prompter string) (string, string, bool, error) {
	// Attempt to probe for a POSIX platform. This might apply to certain
	// Windows environments as well.
	if err := prompt.Message(prompter, "Probing endpoint (POSIX)..."); err != nil {
		return "", "", false, errors.Wrap(err, "unable to message prompter")
	}
	if goos, goarch, err := probePOSIX(transport); err == nil {
		return goos, goarch, true, nil
	}

	// If that fails, attempt a Windows fallback.
	if err := prompt.Message(prompter, "Probing endpoint (Windows)..."); err != nil {
		return "", "", false, errors.Wrap(err, "unable to message prompter")
	}
	if goos, goarch, err := probeWindows(transport); err == nil {
		return goos, goarch, false, nil
	}

	// Failure.
	return "", "", false, errors.New("exhausted probing methods")
}

func install(transport Transport, prompter string) error {
	// Detect the target platform.
	goos, goarch, posix, err := probe(transport, prompter)
	if err != nil {
		return errors.Wrap(err, "unable to probe remote platform")
	}

	// Find the appropriate agent binary. Ensure that it's cleaned up when we're
	// done with it.
	if err := prompt.Message(prompter, "Extracting agent..."); err != nil {
		return errors.Wrap(err, "unable to message prompter")
	}
	agentExecutable, err := executableForPlatform(goos, goarch)
	if err != nil {
		return errors.Wrap(err, "unable to get agent for platform")
	}
	defer os.Remove(agentExecutable)

	// Copy the agent to the remote. We use a unique identifier for the
	// temporary destination. For Windows remotes, we add a ".exe" suffix, which
	// will automatically make the file executable on the remote (POSIX systems
	// are handled separately below). For POSIX systems, we add a dot prefix to
	// hide the executable.
	if err := prompt.Message(prompter, "Copying agent..."); err != nil {
		return errors.Wrap(err, "unable to message prompter")
	}
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return errors.Wrap(err, "unable to generate UUID for agent copying")
	}
	destination := agentBaseName + randomUUID.String()
	if goos == "windows" {
		destination += ".exe"
	}
	if posix {
		destination = "." + destination
	}
	if err = transport.Copy(agentExecutable, destination); err != nil {
		return errors.Wrap(err, "unable to copy agent binary")
	}

	// For cases where we're copying from a Windows system to a POSIX remote,
	// invoke "chmod +x" to add executability back to the copied binary. This is
	// necessary under the specified circumstances because as soon as the agent
	// binary is extracted from the bundle, it will lose its executability bit
	// since Windows can't preserve this. This will also be applied to Windows
	// POSIX remotes, but a "chmod +x" there will just be a no-op.
	if runtime.GOOS == "windows" && posix {
		if err := prompt.Message(prompter, "Setting agent executability..."); err != nil {
			return errors.Wrap(err, "unable to message prompter")
		}
		executabilityCommand := fmt.Sprintf("chmod +x %s", destination)
		if err := run(transport, executabilityCommand); err != nil {
			return errors.Wrap(err, "unable to set agent executability")
		}
	}

	// Invoke the remote installation.
	if err := prompt.Message(prompter, "Installing agent..."); err != nil {
		return errors.Wrap(err, "unable to message prompter")
	}
	var installCommand string
	if posix {
		installCommand = fmt.Sprintf("./%s %s", destination, ModeInstall)
	} else {
		installCommand = fmt.Sprintf("%s %s", destination, ModeInstall)
	}
	if err := run(transport, installCommand); err != nil {
		return errors.Wrap(err, "unable to invoke agent installation")
	}

	// Success.
	return nil
}

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
	connection, err := newConnection(agentProcess)
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
	// TODO: If we decide we want these errors available outside the agent
	// package, it might be worth moving this buffer into the processStream
	// type, exporting that type, and allowing type assertions that would give
	// access to that buffer. But for now we're mostly just concerned with
	// connection issues.
	errorBuffer := bytes.NewBuffer(nil)
	agentProcess.Stderr = errorBuffer

	// Start the process.
	if err = agentProcess.Start(); err != nil {
		return nil, false, false, errors.Wrap(err, "unable to start agent process")
	}

	// Confirm that the process started correctly by performing a version
	// handshake.
	if versionMatch, err := mutagen.ReceiveAndCompareVersion(connection); err != nil {
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

		// If there's an error, check if the command exits with a POSIX "command
		// not found" error, a Windows invalid formatting message (an indication
		// of a cmd.exe environment), or a Windows "command not found" message.
		// We can't really check this until we try to interact with the process
		// and see that it misbehaves. We wouldn't be able to see this returned
		// as an error from the Start method because it just starts the
		// transport command itself, not the remote command.
		if process.IsPOSIXShellCommandNotFound(processErr) {
			return nil, true, false, errors.New("command not found")
		} else if process.OutputIsWindowsInvalidCommand(errorOutput) {
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
	} else if !versionMatch {
		return nil, false, false, errors.New("version mismatch")
	}

	// Wrap the connection in an endpoint client.
	endpoint, err := remote.NewEndpointClient(connection, root, session, version, configuration, alpha)
	if err != nil {
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
