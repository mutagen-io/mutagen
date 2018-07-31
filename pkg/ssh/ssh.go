package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/url"
)

const (
	connectTimeoutSeconds = 5
)

// compressionArgument returns a flag that can be passed to scp or ssh to enable
// compression. Note that while SSH does have a CompressionLevel configuration
// option, this only applies to SSHv1. SSHv2 defaults to a DEFLATE level of 6,
// which is what we want anyway.
func compressionArgument() string {
	return "-C"
}

// timeoutArgument returns a option flag that can be passed to scp or ssh to
// limit connection time (though not transfer time or process lifetime). It is
// currently a fixed value, but in the future we might want to make this
// configurable for people with poor connections.
func timeoutArgument() string {
	return fmt.Sprintf("-oConnectTimeout=%d", connectTimeoutSeconds)
}

// Copy copies a local file (which MUST be an absolute path) to a remote
// destination. If a prompter is provided, this method will attempt to use it
// for authentication if necessary.
func Copy(prompter, local string, remote *url.URL) error {
	// Validate the URL protocol.
	if remote.Protocol != url.Protocol_SSH {
		return errors.New("non-SSH URL provided")
	}

	// Locate the SCP command.
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
	if !filepath.IsAbs(local) {
		return errors.New("scp source path must be absolute")
	}
	workingDirectory, sourceBase := filepath.Split(local)

	// Compute the destination URL.
	destinationURL := fmt.Sprintf("%s:%s", remote.Hostname, remote.Path)
	if remote.Username != "" {
		destinationURL = fmt.Sprintf("%s@%s", remote.Username, destinationURL)
	}

	// Set up arguments.
	var scpArguments []string
	scpArguments = append(scpArguments, compressionArgument())
	scpArguments = append(scpArguments, timeoutArgument())
	if remote.Port != 0 {
		scpArguments = append(scpArguments, "-P", fmt.Sprintf("%d", remote.Port))
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

	// Add prompting environment variables
	environment, err = addPrompterVariables(environment, prompter)
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

// Command create an SSH process set to connect to the specified remote and
// invoke the specified command. This function does not start the process. If a
// prompter is provided, the process will be directed to use it on startup if
// necessary. The command string is interpreted as literal input to the remote
// shell, so its contents are more flexible than just an executable name or
// path. The path component of the remote URL is NOT used as a working directory
// and is simply ignored - the command will execute in whatever default
// directory the server chooses.
func Command(prompter string, remote *url.URL, command string) (*exec.Cmd, error) {
	// Validate the URL protocol.
	if remote.Protocol != url.Protocol_SSH {
		return nil, errors.New("non-SSH URL provided")
	}

	// Locate the SSH command.
	ssh, err := sshCommand()
	if err != nil {
		return nil, errors.Wrap(err, "unable to identify SSH executable")
	}

	// Compute the target.
	target := remote.Hostname
	if remote.Username != "" {
		target = fmt.Sprintf("%s@%s", remote.Username, remote.Hostname)
	}

	// Set up arguments. We intentionally don't use compression on SSH commands
	// since the agent stream uses the FLATE algorithm internally and it's much
	// more efficient to compress at that layer, even with the slower Go
	// implementation.
	var sshArguments []string
	sshArguments = append(sshArguments, timeoutArgument())
	if remote.Port != 0 {
		sshArguments = append(sshArguments, "-p", fmt.Sprintf("%d", remote.Port))
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

	// Add prompting environment variables
	environment, err = addPrompterVariables(environment, prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create prompter environment")
	}

	// Set the environment.
	sshProcess.Env = environment

	// Done.
	return sshProcess, nil
}

// Run creates an SSH command by forwarding its arguments to Command and then
// returning the result of its Run method. If there is an error creating the
// command, it will be returned, but otherwise the result of the Run method will
// be returned un-wrapped, so it can be treated as an os/exec.ExitError.
func Run(prompter string, remote *url.URL, command string) error {
	// Create the process.
	process, err := Command(prompter, remote, command)
	if err != nil {
		return errors.Wrap(err, "unable to create command")
	}

	// Run the process.
	return process.Run()
}

// Output creates an SSH command by forwarding its arguments to Command and then
// returning the results of its Output method. If there is an error creating the
// command, it will be returned, but otherwise the result of the Run method will
// be returned un-wrapped, so it can be treated as an os/exec.ExitError.
func Output(prompter string, remote *url.URL, command string) ([]byte, error) {
	// Create the process.
	process, err := Command(prompter, remote, command)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create command")
	}

	// Run the process.
	return process.Output()
}
