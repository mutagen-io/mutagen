package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/process"
	"github.com/mutagen-io/mutagen/pkg/ssh"
)

const (
	// connectTimeoutSeconds is the number of seconds to use for OpenSSH's
	// ConnectTimeout configuration option.
	connectTimeoutSeconds = 5
	// serverAliveIntervalSeconds is the number of seconds to use for OpenSSH's
	// ServerAliveInterval configuration option. Multiplied by
	// serverAliveCountMax, it effectively limits the maximum allowed latency.
	serverAliveIntervalSeconds = 10
	// serverAliveCountMax is the count to use for OpenSSH's ServerAliveCountMax
	// configuration option.
	serverAliveCountMax = 1
)

// transport implements the agent.Transport interface using SSH.
type transport struct {
	// user is the SSH user under which agents should be invoked.
	user string
	// host is the target host.
	host string
	// port is the target port.
	port uint16
	// prompter is the prompter identifier to use for prompting.
	prompter string
}

// NewTransport creates a new SSH transport using the specified parameters.
func NewTransport(user, host string, port uint16, prompter string) (agent.Transport, error) {
	return &transport{
		user:     user,
		host:     host,
		port:     port,
		prompter: prompter,
	}, nil
}

// Copy implements the Copy method of agent.Transport.
func (t *transport) Copy(localPath, remoteName string) error {
	// HACK: On Windows, we attempt to use SCP executables that might not
	// understand Windows paths because they're designed to run inside a POSIX-
	// style environment (e.g. MSYS or Cygwin). To work around this, we run them
	// in the same directory as the source file and just pass them the source
	// base name. In order to compute the working directory, we need the local
	// path to be absolute, but fortunately this is the case anyway for paths
	// supplied to agent.Transport.Copy. This works fine on non-Windows-POSIX
	// systems as well. We probably don't need this IsAbs sanity check, since
	// path behavior is guaranteed by the Transport interface, but it's better
	// to have as an invariant check.
	if !filepath.IsAbs(localPath) {
		return errors.New("scp source path must be absolute")
	}
	workingDirectory, sourceBase := filepath.Split(localPath)

	// Compute the destination URL.
	// HACK: Since the remote name is supposed to be relative to the user's home
	// directory, we'd ideally want to specify a URL of the form
	// [user@]host:~/remoteName, but the ~/ paradigm isn't understood by
	// Windows. Consequently, we assume that the default destination for SCP
	// copies without a path prefix is the user's home directory, i.e. that the
	// default working directory for the SCP receiving process is the user's
	// home directory. Since we already make the assumption that the home
	// directory is the default working directory for SSH commands, this is a
	// reasonable additional assumption.
	destinationURL := fmt.Sprintf("%s:%s", t.host, remoteName)
	if t.user != "" {
		destinationURL = fmt.Sprintf("%s@%s", t.user, destinationURL)
	}

	// Set up arguments.
	var scpArguments []string
	scpArguments = append(scpArguments, ssh.CompressionFlag())
	scpArguments = append(scpArguments, ssh.ConnectTimeoutFlag(connectTimeoutSeconds))
	scpArguments = append(scpArguments, ssh.ServerAliveFlags(serverAliveIntervalSeconds, serverAliveCountMax)...)
	if t.port != 0 {
		scpArguments = append(scpArguments, "-P", fmt.Sprintf("%d", t.port))
	}
	scpArguments = append(scpArguments, sourceBase, destinationURL)

	// Create the process.
	scpCommand, err := ssh.SCPCommand(context.Background(), scpArguments...)
	if err != nil {
		return errors.Wrap(err, "unable to set up SCP invocation")
	}

	// Set the working directory.
	scpCommand.Dir = workingDirectory

	// Force it to run detached.
	scpCommand.SysProcAttr = process.DetachedProcessAttributes()

	// Create a copy of the current environment.
	environment := os.Environ()

	// Add locale environment variables.
	environment = addLocaleVariables(environment)

	// Set prompting environment variables
	environment, err = SetPrompterVariables(environment, t.prompter)
	if err != nil {
		return errors.Wrap(err, "unable to create prompter environment")
	}

	// Set the environment.
	scpCommand.Env = environment

	// Run the operation.
	if err = scpCommand.Run(); err != nil {
		return errors.Wrap(err, "unable to run SCP process")
	}

	// Success.
	return nil
}

// Command implements the Command method of agent.Transport.
func (t *transport) Command(command string) (*exec.Cmd, error) {
	// Compute the target.
	target := t.host
	if t.user != "" {
		target = fmt.Sprintf("%s@%s", t.user, t.host)
	}

	// Set up arguments. We intentionally don't use compression on SSH commands
	// since the agent stream uses the FLATE algorithm internally and it's much
	// more efficient to compress at that layer, even with the slower Go
	// implementation.
	var sshArguments []string
	sshArguments = append(sshArguments, ssh.ConnectTimeoutFlag(connectTimeoutSeconds))
	sshArguments = append(sshArguments, ssh.ServerAliveFlags(serverAliveIntervalSeconds, serverAliveCountMax)...)
	if t.port != 0 {
		sshArguments = append(sshArguments, "-p", fmt.Sprintf("%d", t.port))
	}
	sshArguments = append(sshArguments, target, command)

	// Create the process.
	sshCommand, err := ssh.SSHCommand(context.Background(), sshArguments...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to set up SSH invocation")
	}

	// Force it to run detached.
	sshCommand.SysProcAttr = process.DetachedProcessAttributes()

	// Create a copy of the current environment.
	environment := os.Environ()

	// Add locale environment variables.
	environment = addLocaleVariables(environment)

	// Set prompting environment variables
	environment, err = SetPrompterVariables(environment, t.prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create prompter environment")
	}

	// Set the environment.
	sshCommand.Env = environment

	// Done.
	return sshCommand, nil
}

// ClassifyError implements the ClassifyError method of agent.Transport.
func (t *transport) ClassifyError(processState *os.ProcessState, errorOutput string) (bool, bool, error) {
	// SSH faithfully returns exit codes and error output, so we can use direct
	// methods for testing and classification. Note that we may get POSIX-like
	// error codes back even from Windows remotes, but that indicates a POSIX
	// shell on the remote and thus we should continue connecting under that
	// hypothesis (instead of the cmd.exe hypothesis).
	if process.IsPOSIXShellInvalidCommand(processState) {
		return true, false, nil
	} else if process.IsPOSIXShellCommandNotFound(processState) {
		return true, false, nil
	} else if process.OutputIsWindowsInvalidCommand(errorOutput) {
		// A Windows invalid command error doesn't necessarily indicate that
		// the agent isn't installed, but instead usually indicates that we were
		// trying to invoke the agent using the POSIX shell syntax in a Windows
		// cmd.exe environment. Thus we return false here for re-installation,
		// but we still indicate that this is a Windows platform to potentially
		// change the dialer's platform hypothesis and force it to reconnect
		// under the Windows hypothesis.
		// HACK: We're relying on the fact that the agent dialing logic will
		// attempt a reconnect under the cmd.exe hypothesis, which it will, but
		// this is potentially a bit fragile. We've sort of codified this
		// behavior in the transport interface definition, but it's hard to make
		// super explicit.
		return false, true, nil
	} else if process.OutputIsWindowsCommandNotFound(errorOutput) {
		return true, true, nil
	}

	// Just bail if we weren't able to determine the nature of the error.
	return false, false, errors.New("unknown error condition encountered")
}
