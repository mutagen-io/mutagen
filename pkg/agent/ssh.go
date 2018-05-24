package agent

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/google/uuid"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/process"
	"github.com/havoc-io/mutagen/pkg/ssh"
	"github.com/havoc-io/mutagen/pkg/url"
)

const (
	posixCommandNotFoundExitCode   = 127
	windowsInvalidCommandFragment  = "is not recognized as an internal or external command"
	windowsCommandNotFoundFragment = "The system cannot find the path specified"
)

func isPOSIXCommandNotFound(err error) bool {
	code, codeErr := process.ExitCodeForError(err)
	return codeErr == nil && code == posixCommandNotFoundExitCode
}

func probeSSHPOSIX(remote *url.URL, prompter string) (string, string, error) {
	// Try to invoke uname and print kernel and machine name.
	unameSMBytes, err := ssh.Output(prompter, "Probing endpoint", remote, "uname -s -m")
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

func probeSSHWindows(remote *url.URL, prompter string) (string, string, error) {
	// Attempt to dump the remote environment.
	outputBytes, err := ssh.Output(prompter, "Probing endpoint", remote, "cmd /c set")
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

// probeSSHPlatform attempts to identify the properties of the target platform,
// namely GOOS, GOARCH, and whether or not it's a POSIX environment (which it
// might be even on Windows).
func probeSSHPlatform(remote *url.URL, prompter string) (string, string, bool, error) {
	// Attempt to probe for a POSIX platform. This might apply to certain
	// Windows environments as well.
	if goos, goarch, err := probeSSHPOSIX(remote, prompter); err == nil {
		return goos, goarch, true, nil
	}

	// If that fails, attempt a Windows fallback.
	if goos, goarch, err := probeSSHWindows(remote, prompter); err == nil {
		return goos, goarch, false, nil
	}

	// Failure.
	return "", "", false, errors.New("exhausted probing methods")
}

func installSSH(remote *url.URL, prompter string) error {
	// Detect the target platform.
	goos, goarch, posix, err := probeSSHPlatform(remote, prompter)
	if err != nil {
		return errors.Wrap(err, "unable to probe remote platform")
	}

	// Find the appropriate agent binary. Ensure that it's cleaned up when we're
	// done with it.
	agent, err := executableForPlatform(goos, goarch)
	if err != nil {
		return errors.Wrap(err, "unable to get agent for platform")
	}
	defer os.Remove(agent)

	// Copy the agent to the remote. We use a unique identifier for the
	// temporary destination. For Windows remotes, we add a ".exe" suffix, which
	// will automatically make the file executable on the remote (POSIX systems
	// are handled separately below). For POSIX systems, we add a dot prefix to
	// hide the executable.
	// HACK: This assumes that the SSH user's home directory is used as the
	// default destination directory for SCP copies. That should be true in
	// 99.9% of cases, but if it becomes a major issue, we'll need to use the
	// probe information to handle this more carefully.
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
	destinationURL := &url.URL{
		Protocol: remote.Protocol,
		Username: remote.Username,
		Hostname: remote.Hostname,
		Port:     remote.Port,
		Path:     destination,
	}
	if err := ssh.Copy(prompter, "Copying agent", agent, destinationURL); err != nil {
		return errors.Wrap(err, "unable to copy agent binary")
	}

	// For cases where we're copying from a Windows system to a POSIX remote,
	// invoke "chmod +x" to add executability back to the copied binary. This is
	// necessary under the specified circumstances because as soon as the agent
	// binary is extracted from the bundle, it will lose its executability bit
	// since Windows can't preserve this. This will also be applied to Windows
	// POSIX remotes, but a "chmod +x" there will just be a no-op.
	// HACK: This assumes that the SSH user's home directory is used as the
	// default working directory for SSH commands. We have to do this because we
	// don't have a portable mechanism to invoke the command relative to the
	// user's home directory and we don't want to do a probe of the remote
	// system before invoking the command. This assumption should be fine for
	// 99.9% of cases, but if it becomes a major issue, we'll need to use the
	// probe information to handle this more carefully.
	if runtime.GOOS == "windows" && posix {
		executabilityCommand := fmt.Sprintf("chmod +x %s", destination)
		if err := ssh.Run(prompter, "Setting agent executability", remote, executabilityCommand); err != nil {
			return errors.Wrap(err, "unable to set agent executability")
		}
	}

	// Invoke the remote installation.
	// HACK: This assumes that the SSH user's home directory is used as the
	// default working directory for SSH commands. The reasons for assuming this
	// are outlined above.
	var installCommand string
	if posix {
		installCommand = fmt.Sprintf("./%s install", destination)
	} else {
		installCommand = fmt.Sprintf("%s install", destination)
	}
	if err := ssh.Run(prompter, "Installing agent", remote, installCommand); err != nil {
		return errors.Wrap(err, "unable to invoke agent installation")
	}

	// Success.
	return nil
}

func connectSSH(remote *url.URL, prompter, mode string, windows bool) (net.Conn, bool, bool, error) {
	// Compute the agent SSH command. Unless we have reason to assume that this
	// is a Windows system, we construct a path using forward slashes. This will
	// work for all POSIX systems and POSIX-like environments on Windows. If we
	// know we're hitting a Windows SSH server that's going to invoke commands
	// inside cmd.exe, then use backslashes, otherwise the invocation won't
	// work.
	// HACK: This assumes that the SSH user's home directory is used as the
	// default working directory for SSH commands. We have to do this because we
	// don't have a portable mechanism to invoke the command relative to the
	// user's home directory (tilde doesn't work on Windows) and we don't want
	// to do a probe of the remote system before invoking the endpoint. This
	// assumption should be fine for 99.9% of cases, but if it becomes a major
	// issue, the only other options I see are probing before invoking (slow) or
	// using the Go SSH library to do this (painful to faithfully emulate
	// OpenSSH's behavior). Perhaps we can allow expicit home directory
	// specification via a command line flag if this becomes necessary.
	// HACK: We're assuming that none of these path components have spaces in
	// them, but since we control all of them, this is probably okay.
	// HACK: When invoking on Windows systems (whether inside a POSIX
	// environment or cmd.exe), we can leave the "exe" suffix off the target
	// name. Fortunately this allows us to also avoid having to try the
	// combination of forward slashes + ".exe" for Windows POSIX environments.
	pathSeparator := "/"
	if windows {
		pathSeparator = "\\"
	}
	sshAgentPath := strings.Join([]string{
		filesystem.MutagenDirectoryName,
		agentsDirectoryName,
		mutagen.Version,
		agentBaseName,
	}, pathSeparator)

	// Compute the command to invoke.
	// HACK: We rely on sshAgentPath not having any spaces in it. If we do
	// eventually need to add any, we'll need to fix this up for the shell.
	command := fmt.Sprintf("%s %s", sshAgentPath, mode)

	// Create an SSH process.
	process, err := ssh.Command(prompter, "Connecting to agent", remote, command)
	if err != nil {
		return nil, false, false, errors.Wrap(err, "unable to create SSH command")
	}

	// Create a connection that wrap's the process' standard input/output.
	connection, err := newAgentConnection(remote, process)
	if err != nil {
		return nil, false, false, errors.Wrap(err, "unable to create SSH process connection")
	}

	// Redirect the process' standard error output to a buffer so that we can
	// give better feedback in errors. This might be a bit dangerous since this
	// buffer will be attached for the lifetime of the process and we don't know
	// exactly how much output will be received (and thus we could buffer a
	// large amount of it in memory), but generally speaking SSH doens't spit
	// out much error output (unless in debug mode, which we won't be), and the
	// agent doesn't spit out any.
	// TODO: If we do start seeing large allocations in these buffers, a simple
	// size-limited buffer might suffice, at least to get some of the error
	// message.
	// TODO: If we decide we want these errors available outside the agent
	// package, it might be worth moving this buffer into the processStream
	// type, exporting that type, and allowing type assertions that would give
	// access to that buffer. But for now we're mostly just concerned with
	// connection issues.
	errorBuffer := bytes.NewBuffer(nil)
	process.Stderr = errorBuffer

	// Start the process.
	if err = process.Start(); err != nil {
		return nil, false, false, errors.Wrap(err, "unable to start SSH agent process")
	}

	// Confirm that the process started correctly by performing a version
	// handshake.
	if versionMatch, err := mutagen.ReceiveAndCompareVersion(connection); err != nil {
		// Extract error output and ensure it's UTF-8.
		errorOutput := errorBuffer.String()
		if !utf8.ValidString(errorOutput) {
			return nil, false, false, errors.New("remote did not return UTF-8 output")
		}

		// If there's an error, check if SSH exits with a command not found
		// error (or an error message on Windows indicating and invalid command
		// (due to POSIX path formatting) or command not found). We can't really
		// check this until we try to interact with the process and see that it
		// misbehaves. We wouldn't be able to see this returned as an error from
		// the Start method because it just starts the SSH client itself, not
		// the remote command.
		if isPOSIXCommandNotFound(process.Wait()) {
			return nil, true, false, errors.New("command not found")
		} else if strings.Index(errorOutput, windowsInvalidCommandFragment) != -1 {
			return nil, true, true, errors.New("invalid command")
		} else if strings.Index(errorOutput, windowsCommandNotFoundFragment) != -1 {
			return nil, true, true, errors.New("command not found")
		}

		// Otherwise, check if there is any error output that might illuminate
		// what happened. We let this overrule any err value here since that
		// value will probably just be an EOF.
		if errorOutput != "" {
			return nil, false, false, errors.Errorf(
				"SSH process failed with error output:\n%s",
				strings.TrimSpace(errorOutput),
			)
		}

		// Otherwise just wrap up whatever error we have.
		return nil, false, false, errors.Wrap(err, "unable to handshake with SSH agent process")
	} else if !versionMatch {
		return nil, false, false, errors.New("version mismatch")
	}

	// Done.
	return connection, false, false, nil
}

func DialSSH(remote *url.URL, prompter, mode string) (net.Conn, error) {
	// Attempt a connection. If this fails but we detect a Windows environment
	// in the process, then re-attempt a connection under the Windows
	// assumption.
	connection, install, windows, err := connectSSH(remote, prompter, mode, false)
	if err == nil {
		return connection, nil
	} else if windows {
		connection, install, windows, err = connectSSH(remote, prompter, mode, true)
		if err == nil {
			return connection, nil
		}
	}

	// If we didn't detect a Windows environment, or we did and it also failed,
	// then check if an install is recommended. If not, then bail.
	if !install {
		return nil, errors.Wrap(err, "unable to connect to agent")
	}

	// Attempt to install.
	if err := installSSH(remote, prompter); err != nil {
		return nil, errors.Wrap(err, "unable to install agent")
	}

	// Re-attempt connectivity.
	if connection, _, _, err := connectSSH(remote, prompter, mode, windows); err != nil {
		return nil, errors.Wrap(err, "unable to connect to agent")
	} else {
		return connection, nil
	}
}
