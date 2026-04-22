package ssh

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/agent/transport"
	"github.com/mutagen-io/mutagen/pkg/process"
	"github.com/mutagen-io/mutagen/pkg/ssh"
)

const (
	// serverAliveIntervalSeconds is the number of seconds to use for OpenSSH's
	// ServerAliveInterval configuration option. Multiplied by
	// serverAliveCountMax, it effectively limits the maximum allowed latency.
	serverAliveIntervalSeconds = 10
	// serverAliveCountMax is the count to use for OpenSSH's ServerAliveCountMax
	// configuration option.
	serverAliveCountMax = 1
)

var (
	// connectTimeoutSeconds is the number of seconds to use for OpenSSH's
	// ConnectTimeout configuration option.
	connectTimeoutSeconds uint64 = 5
)

func init() {
	// If a valid connection timeout has been specified in the environment, then
	// override the default connection timeout setting.
	if t, err := strconv.ParseUint(os.Getenv("MUTAGEN_SSH_CONNECT_TIMEOUT"), 10, 64); err == nil && t > 0 {
		connectTimeoutSeconds = t
	}
}

// sshTransport implements the agent.Transport interface using SSH.
type sshTransport struct {
	// user is the SSH user under which agents should be invoked.
	user string
	// host is the target host.
	host string
	// port is the target port.
	port uint16
	// prompter is the prompter identifier to use for prompting.
	prompter string
	// configPath is the path to the SSH config file to use, if specified.
	configPath string
}

// NewTransport creates a new SSH transport using the specified parameters.
func NewTransport(user, host string, port uint16, prompter, configPath string) (agent.Transport, error) {
	return &sshTransport{
		user:       user,
		host:       host,
		port:       port,
		prompter:   prompter,
		configPath: configPath,
	}, nil
}

// Copy implements the Copy method of agent.Transport.
func (t *sshTransport) Copy(localPath, remoteName string) error {
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
	scpArguments = append(scpArguments, ssh.ConfigFlags(t.configPath)...)
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
		return fmt.Errorf("unable to set up SCP invocation: %w", err)
	}

	// Set the working directory.
	scpCommand.Dir = workingDirectory

	// Set the process attributes.
	scpCommand.SysProcAttr = transport.ProcessAttributes()

	// Compute the default environment for the process.
	environment := scpCommand.Environ()

	// Add locale environment variables.
	environment = addLocaleVariables(environment)

	// Set prompting environment variables
	environment, err = SetPrompterVariables(environment, t.prompter)
	if err != nil {
		return fmt.Errorf("unable to create prompter environment: %w", err)
	}

	// Set the environment.
	scpCommand.Env = environment

	// Run the operation.
	if _, err = scpCommand.Output(); err != nil {
		if message := process.ExtractExitErrorMessage(err); message != "" {
			return fmt.Errorf("unable to run SCP process: %s", message)
		}
		return fmt.Errorf("unable to run SCP process: %w", err)
	}

	// Success.
	return nil
}

// Command implements the Command method of agent.Transport.
func (t *sshTransport) Command(command string) (*exec.Cmd, error) {
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
	sshArguments = append(sshArguments, ssh.ConfigFlags(t.configPath)...)
	sshArguments = append(sshArguments, ssh.ConnectTimeoutFlag(connectTimeoutSeconds))
	sshArguments = append(sshArguments, ssh.ServerAliveFlags(serverAliveIntervalSeconds, serverAliveCountMax)...)
	if t.port != 0 {
		sshArguments = append(sshArguments, "-p", fmt.Sprintf("%d", t.port))
	}
	sshArguments = append(sshArguments, target, command)

	// Create the process.
	sshCommand, err := ssh.SSHCommand(context.Background(), sshArguments...)
	if err != nil {
		return nil, fmt.Errorf("unable to set up SSH invocation: %w", err)
	}

	// Force it to run detached.
	sshCommand.SysProcAttr = transport.ProcessAttributes()

	// Compute the default environment for the process.
	environment := sshCommand.Environ()

	// Add locale environment variables.
	environment = addLocaleVariables(environment)

	// Set prompting environment variables
	environment, err = SetPrompterVariables(environment, t.prompter)
	if err != nil {
		return nil, fmt.Errorf("unable to create prompter environment: %w", err)
	}

	// Set the environment.
	sshCommand.Env = environment

	// Done.
	return sshCommand, nil
}

// ClassifyError implements the ClassifyError method of agent.Transport.
func (t *sshTransport) ClassifyError(processState *os.ProcessState, errorOutput string) (bool, bool, error) {
	// SSH faithfully returns exit codes and error output, so we can use direct
	// methods for testing and classification. Note that we may get POSIX-like
	// error codes back even from Windows remotes, but that indicates a POSIX
	// shell on the remote and thus we should continue connecting under that
	// hypothesis (instead of the cmd.exe hypothesis).
	//
	// NOTE: If advanced SSH options (such as ProxyCommand) are used, then it's
	// possible that we might misclassify command-not-found errors returned by
	// SSH itself as command-not-found errors returned by the remote. In this
	// case, such errors would never be shown to the user, because we would
	// immediately move to install an agent binary. To mitigate this problem, we
	// capture and return scp error output above as part of any Copy error,
	// which will help to inform the user of any SSH transport issues arising
	// from the SSH configuration.
	if process.IsPOSIXShellInvalidCommand(processState) {
		return true, false, nil
	} else if process.IsPOSIXShellCommandNotFound(processState) {
		return true, false, nil
	} else if process.OutputIsPOSIXCommandNotFound(errorOutput) {
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
