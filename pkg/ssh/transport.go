package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/process"
	"github.com/havoc-io/mutagen/pkg/url"
)

// transport implements the agent.Transport interface using SSH.
type transport struct {
	// remote is the endpoint URL.
	remote *url.URL
	// prompter is the prompter identifier to use for prompting.
	prompter string
}

// Copy implements the Copy method of agent.Transport.
func (t *transport) Copy(localPath, remoteName string) error {
	// Locate the SCP command.
	// TODO: Should we cache this inside the transport? Even if the user changes
	// their path or adds something to it, we probably wouldn't pick it up until
	// the daemon restarts, so I'm not sure there's a point to recomputing this
	// every time. Well, actually, we could pick it up on Windows, which is
	// likely where it would matter most. Perhaps we should even pre-compute it
	// when we construct the transport, but then we need to compute it at
	// construction (i.e. startup) time, and there's not really a good way to
	// handle errors at that point.
	scp, err := scpCommand()
	if err != nil {
		return errors.Wrap(err, "unable to identify SCP executable")
	}

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
	destinationURL := fmt.Sprintf("%s:%s", t.remote.Hostname, remoteName)
	if t.remote.Username != "" {
		destinationURL = fmt.Sprintf("%s@%s", t.remote.Username, destinationURL)
	}

	// Set up arguments.
	var scpArguments []string
	scpArguments = append(scpArguments, compressionArgument())
	scpArguments = append(scpArguments, timeoutArgument())
	if t.remote.Port != 0 {
		scpArguments = append(scpArguments, "-P", fmt.Sprintf("%d", t.remote.Port))
	}
	scpArguments = append(scpArguments, sourceBase, destinationURL)

	// Create the process.
	scpProcess := exec.Command(scp, scpArguments...)

	// Set the working directory.
	scpProcess.Dir = workingDirectory

	// Force it to run detached.
	scpProcess.SysProcAttr = process.DetachedProcessAttributes()

	// Create a copy of the current environment.
	environment := os.Environ()

	// Add locale environment variables.
	environment = addLocaleVariables(environment)

	// Set prompting environment variables
	environment, err = setPrompterVariables(environment, t.prompter)
	if err != nil {
		return errors.Wrap(err, "unable to create prompter environment")
	}

	// Set the environment.
	scpProcess.Env = environment

	// Run the operation.
	if err = scpProcess.Run(); err != nil {
		return errors.Wrap(err, "unable to run SCP process")
	}

	// Success.
	return nil
}

// Command implements the Command method of agent.Transport.
func (t *transport) Command(command string) (*exec.Cmd, error) {
	// Locate the SSH command.
	// TODO: Should we cache this inside the transport? Even if the user changes
	// their path or adds something to it, we probably wouldn't pick it up until
	// the daemon restarts, so I'm not sure there's a point to recomputing this
	// every time. Well, actually, we could pick it up on Windows, which is
	// likely where it would matter most. Perhaps we should even pre-compute it
	// when we construct the transport, but then we need to compute it at
	// construction (i.e. startup) time, and there's not really a good way to
	// handle errors at that point.
	ssh, err := sshCommand()
	if err != nil {
		return nil, errors.Wrap(err, "unable to identify SSH executable")
	}

	// Compute the target.
	target := t.remote.Hostname
	if t.remote.Username != "" {
		target = fmt.Sprintf("%s@%s", t.remote.Username, t.remote.Hostname)
	}

	// Set up arguments. We intentionally don't use compression on SSH commands
	// since the agent stream uses the FLATE algorithm internally and it's much
	// more efficient to compress at that layer, even with the slower Go
	// implementation.
	var sshArguments []string
	sshArguments = append(sshArguments, timeoutArgument())
	if t.remote.Port != 0 {
		sshArguments = append(sshArguments, "-p", fmt.Sprintf("%d", t.remote.Port))
	}
	sshArguments = append(sshArguments, target, command)

	// Create the process.
	sshProcess := exec.Command(ssh, sshArguments...)

	// Force it to run detached.
	sshProcess.SysProcAttr = process.DetachedProcessAttributes()

	// Create a copy of the current environment.
	environment := os.Environ()

	// Add locale environment variables.
	environment = addLocaleVariables(environment)

	// Set prompting environment variables
	environment, err = setPrompterVariables(environment, t.prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create prompter environment")
	}

	// Set the environment.
	sshProcess.Env = environment

	// Done.
	return sshProcess, nil
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
