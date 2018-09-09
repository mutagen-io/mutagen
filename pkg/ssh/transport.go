package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

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
func (t *transport) Copy(localPath, remotePath string) error {
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
	// in the same directory as the source and just pass them the source base
	// name. This works fine on other systems as well. Unfortunately this means
	// that we need to use absolute paths, but we do that anyway.
	if !filepath.IsAbs(localPath) {
		return errors.New("scp source path must be absolute")
	}
	workingDirectory, sourceBase := filepath.Split(localPath)

	// Compute the destination URL.
	destinationURL := fmt.Sprintf("%s:%s", t.remote.Hostname, remotePath)
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
	scpProcess.SysProcAttr = processAttributes()

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
	sshProcess.SysProcAttr = processAttributes()

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
