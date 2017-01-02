package agent

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/environment"
	"github.com/havoc-io/mutagen/filesystem"
	"github.com/havoc-io/mutagen/ssh"
	"github.com/havoc-io/mutagen/url"
)

var sshAgentPath string

func init() {
	// Compute the agent SSH command.
	// HACK: This assumes that the SSH user's home directory is used as the
	// default working directory for SSH commands. We have to do this because we
	// don't have a portable mechanism to invoke the command relative to the
	// user's home directory (tilde doesn't work on Windows) and we don't want
	// to do a probe of the remote system before invoking the endpoint. This
	// assumption should be fine for 99.9% of cases, but if it becomes a major
	// issue, the only other options I see are probing before invoking (slow) or
	// using the Go SSH library to do this (painful to faithfully emulate
	// OpenSSH's behavior). Perhaps probing could be hidden behind an option?
	// HACK: We're assuming that none of these path components have spaces in
	// them, but since we control all of them, this is probably okay.
	// HACK: When invoking on Windows systems, we can use forward slashes for
	// the path and leave the "exe" suffix off the target name. This saves us a
	// target check.
	sshAgentPath = path.Join(
		filesystem.MutagenDirectoryName,
		agentsDirectoryName,
		mutagen.Version,
		agentBaseName,
	)
}

func probeSSHPOSIX(remote *url.URL, prompter string) (string, string, error) {
	// Try to invoke uname and print kernel and machine name.
	unameSMBytes, err := ssh.Output(prompter, "Probing endpoint", remote, "uname -s -m")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to invoke uname")
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
	// Try to print the remote environment.
	envBytes, err := ssh.Output(prompter, "Probing endpoint", remote, "cmd /c set")
	if err != nil {
		return "", "", errors.Wrap(err, "unable to invoke set")
	}

	// Parse set output.
	env, err := environment.ParseBlock(string(envBytes))
	if err != nil {
		return "", "", errors.Wrap(err, "unable to parse environment")
	}

	// Translate GOOS.
	goos, ok := osEnvToGOOS[env["OS"]]
	if !ok {
		return "", "", errors.New("unknown platform")
	}

	// Translate GOARCH.
	goarch, ok := processorArchitectureEnvToGOARCH[env["PROCESSOR_ARCHITECTURE"]]
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
	// hide the executable a bit.
	// HACK: This assumes that the SSH user's home directory is used as the
	// default destination directory for SCP copies. That should be true in
	// 99.9% of cases, but if it becomes a major issue, we'll need to use the
	// probe information to handle this more carefully.
	destination := agentBaseName + uuid.NewV4().String()
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

	// Invoke the remote installation. For POSIX remotes, we have to incorporate
	// a "chmod +x" in order for the remote to execute the installer. The POSIX
	// solution is necessary in the event that an installer is sent from a
	// Windows to a POSIX system using SCP, since there's no way to preserve the
	// executability bit (Windows doesn't have one). This will also be applied
	// to Windows POSIX environments, but a "chmod +x" there will have no
	// effect.
	// HACK: This assumes that the SSH user's home directory is used as the
	// default working directory for SSH commands. We have to do this because we
	// don't have a portable mechanism to invoke the command relative to the
	// user's home directory and we don't want to do a probe of the remote
	// system before invoking the endpoint. This assumption should be fine for
	// 99.9% of cases, but if it becomes a major issue, we'll need to use the
	// probe information to handle this more carefully.
	var installCommand string
	if posix {
		installCommand = fmt.Sprintf("chmod +x %s && ./%s install", destination, destination)
	} else {
		installCommand = fmt.Sprintf("%s install", destination)
	}
	if err := ssh.Run(prompter, "Installing agent", remote, installCommand); err != nil {
		return errors.Wrap(err, "unable to invoke agent installation")
	}

	// Success.
	return nil
}

func connectSSH(remote *url.URL, prompter, mode string) (io.ReadWriteCloser, bool, error) {
	// Compute the command to invoke.
	// HACK: We rely on sshAgentPath not having any spaces in it. If we do
	// eventually need to add any, we'll need to fix this up for the shell.
	command := fmt.Sprintf("%s %s", sshAgentPath, mode)

	// Create an SSH process.
	process, err := ssh.Command(prompter, "Connecting to agent", remote, command)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to create SSH command")
	}

	// Create a stream that wrap's the process' standard input/output.
	stream, err := newProcessStream(process)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to create SSH process stream")
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
		return nil, false, errors.Wrap(err, "unable to start SSH agent process")
	}

	// Confirm that the process started correctly by performing a version
	// handshake.
	if versionMatch, err := mutagen.ReceiveAndCompareVersion(stream); err != nil {
		// If there's an error, check if SSH exits with a command not found
		// error. We can't really check this until we try to interact with the
		// process and see that it misbehaves. We wouldn't be able to see this
		// returned as an error from the Start method because it just starts the
		// SSH client itself, not the remote command.
		if ssh.IsCommandNotFound(process.Wait()) {
			return nil, true, errors.New("command not found")
		}

		// Otherwise, check if there is any error output that might illuminate
		// what happened. We let this overrule any err value here since that
		// value will probably just be an EOF.
		if errorBuffer.Len() > 0 {
			return nil, false, errors.Errorf(
				"SSH process failed with error output:\n%s",
				strings.TrimSpace(errorBuffer.String()),
			)
		}

		// Otherwise just wrap up whatever error we have.
		return nil, false, errors.Wrap(err, "unable to handshake with SSH agent process")
	} else if !versionMatch {
		return nil, true, errors.New("version mismatch")
	}

	// Create a connection.
	return stream, false, nil
}

func DialSSH(remote *url.URL, prompter, mode string) (io.ReadWriteCloser, error) {
	// Attempt a connection. If this fails, but it's a failure that justfies
	// attempting an install, then continue, otherwise fail.
	if connection, install, err := connectSSH(remote, prompter, mode); err == nil {
		return connection, nil
	} else if !install {
		return nil, errors.Wrap(err, "unable to connect to agent")
	}

	// Attempt to install.
	if err := installSSH(remote, prompter); err != nil {
		return nil, errors.Wrap(err, "unable to install agent")
	}

	// Re-attempt connectivity.
	if connection, _, err := connectSSH(remote, prompter, mode); err != nil {
		return nil, errors.Wrap(err, "unable to connect to agent")
	} else {
		return connection, nil
	}
}
