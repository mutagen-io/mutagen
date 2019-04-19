package kubectl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/process"
	"github.com/havoc-io/mutagen/pkg/url"
)

// transport implements the agent.Transport interface using Kubectl.
type transport struct {
	// kubectlExecutable is the name of or path to the kubectl executable.
	kubectlExecutable string
	// remote is the endpoint URL.
	remote *url.URL
	// prompter is the prompter identifier to use for prompting.
	prompter string
	// containerProbed indicates whether or not container probing has occurred.
	// If true, then either containerHomeDirectory will be non-empty or
	// containerProbeError will be non-nil.
	containerProbed bool
	// containerIsWindows indicates whether or not the container is a Windows
	// container. If not, it should be assumed that it is a POSIX (effectively
	// Linux) container.
	containerIsWindows bool
	// containerHomeDirectory is the path to the specified user's home directory
	// within the container.
	containerHomeDirectory string
	// containerProbeError tracks any error that arose when probing the
	// container.
	containerProbeError error
}

// newTransport creates a new Kubectl transport.
func newTransport(remote *url.URL, prompter string) (*transport, error) {
	// Identify the name of or path to the Kubectl executable.
	kubectlExecutable, err := kubectlCommand()
	if err != nil {
		return nil, errors.Wrap(err, "unable to locate kubectl executable")
	}

	// Success.
	return &transport{
		kubectlExecutable: kubectlExecutable,
		remote:            remote,
		prompter:          prompter,
	}, nil
}

// command executes provided command inside default pod using kubectl exec -i
// If command starts with "./" then we replace "." with default users home directory
// Else we just run command as is.
// This makes it mutagen agent to execute file relative to home directory
// without actualy knowing the location of home directory
func (t *transport) command(command string) *exec.Cmd {
	// Tell Kubectl that we want to execute a command in an interactive (i.e.
	// with standard input attached) fashion.
	kubectlArguments := []string{"exec", "-i"}

	// Set the pod name (this is stored as the Hostname field in the URL).
	// TODO: Add support to specify container
	kubectlArguments = append(kubectlArguments, t.remote.Hostname)

	// Split kubectl arguments from exec command arguments
	kubectlArguments = append(kubectlArguments, "--")

	// In case command starts with "./", we replace the "." with default users home directory.
	// This way we have predictabile location of file to be executed. Always releative to default users home directory.
	// Copying files is also relative to this directory. Which ensures we can copy and then access the agent.
	if strings.HasPrefix(command, "./") {
		command = fmt.Sprintf("%s/%s",
			t.containerHomeDirectory,
			strings.TrimPrefix(command, "."),
		)
	}

	// Lex the command that we want to run since Kubectl, unlike SSH, wants the
	// commands and arguments separately instead of as a single argument. All
	// agent.Transport interfaces only need to support commands that can be
	// lexed by splitting on spaces, so we don't need to pull in a more complex
	// shell lexing package here.
	kubectlArguments = append(kubectlArguments, strings.Split(command, " ")...)

	// Create the command.
	kubectlCommand := exec.Command(t.kubectlExecutable, kubectlArguments...)

	// Force it to run detached.
	kubectlCommand.SysProcAttr = process.DetachedProcessAttributes()

	// Create a copy of the current environment.
	environment := os.Environ()

	// Set Kubectl environment variables.
	environment = setKubectlVariables(environment, t.remote)

	// Set the environment for the command.
	kubectlCommand.Env = environment

	// Done.
	return kubectlCommand
}

// probeContainer ensures that the containerIsWindows and containerHomeDirectory
// fields are populated. It is idempotent. If probing previously failed, probing
// will simply return an error indicating the previous failure.
func (t *transport) probeContainer() error {
	// Watch for previous errors.
	if t.containerProbeError != nil {
		return errors.Wrap(t.containerProbeError, "previous container probing failed")
	}

	// Check if we've already probed. If not, then we're going to probe, so mark
	// it as complete (even if it isn't ultimately successful).
	if t.containerProbed {
		return nil
	}
	t.containerProbed = true

	// Track what we've discovered so far in our probes.
	var windows bool
	var home string
	var posixErr, windowsErr error

	// Attempt to run env in the container to probe the user's environment on
	// POSIX systems and identify the HOME environment variable value. If we
	// detect a non-UTF-8 output or detect an empty home directory, we treat
	// that as an error.
	if envBytes, err := t.command("env").Output(); err == nil {
		if !utf8.Valid(envBytes) {
			t.containerProbeError = errors.New("non-UTF-8 POSIX environment")
			return t.containerProbeError
		} else if h, ok := findEnviromentVariable(string(envBytes), "HOME"); ok {
			if h == "" {
				t.containerProbeError = errors.New("empty POSIX home directory")
				return t.containerProbeError
			}
			home = h
		}
	} else {
		posixErr = err
	}

	// If we didn't find a POSIX home directory, attempt to a similar procedure
	// on Windows to identify the USERPROFILE environment variable.
	if home == "" {
		if envBytes, err := t.command("cmd /c set").Output(); err == nil {
			if !utf8.Valid(envBytes) {
				t.containerProbeError = errors.New("non-UTF-8 Windows environment")
				return t.containerProbeError
			} else if h, ok := findEnviromentVariable(string(envBytes), "USERPROFILE"); ok {
				if h == "" {
					t.containerProbeError = errors.New("empty Windows home directory")
					return t.containerProbeError
				}
				home = h
				windows = true
			}
		} else {
			windowsErr = err
		}
	}

	// If both probing mechanisms have failed, then create a combined error
	// message. This is a bit verbose, but it's the only way to get out all of
	// the information that we need. We could prioritize POSIX errors over
	// Windows errors, but that would effectively always mask Windows errors due
	// to the fact that we'd get a "command not found" error when trying to run
	// env on Windows, and we'd never see what error arose on the Windows side.
	if home == "" {
		t.containerProbeError = errors.Errorf(
			"container probing failed under POSIX hypothesis (%s) and Windows hypothesis (%s)",
			posixErr.Error(),
			windowsErr.Error(),
		)
		return t.containerProbeError
	}

	// Store values.
	t.containerIsWindows = windows
	t.containerHomeDirectory = home

	// Success.
	return nil
}

// Copy implements the Copy method of agent.Transport.
func (t *transport) Copy(localPath, remoteName string) error {
	// Ensure that the container has been probed.
	if err := t.probeContainer(); err != nil {
		return errors.Wrap(err, "unable to probe container")
	}

	// Compute the path inside the container. We don't bother trimming trailing
	// slashes from the home directory, because both Windows and POSIX will work
	// in their presence. The only case on Windows where \\ has special meaning
	// is with UNC paths, an in that case they only occur at the beginning of a
	// path, which they won't in this case since we've verified that the home
	// directory is non-empty.
	//
	// We also check if path starts with / or \\. In case it is, we dont change it.
	// But if it dose, then we prepend default users home directory.
	var containerPath string
	if t.containerIsWindows {
		if strings.HasPrefix(remoteName, "\\") {
			containerPath = fmt.Sprintf("%s:%s",
				t.remote.Hostname,
				remoteName,
			)
		} else {
			containerPath = fmt.Sprintf("%s:%s\\%s",
				t.remote.Hostname,
				t.containerHomeDirectory,
				remoteName,
			)
		}
	} else {
		if strings.HasPrefix(remoteName, "/") {
			containerPath = fmt.Sprintf("%s:%s",
				t.remote.Hostname,
				remoteName,
			)
		} else {
			containerPath = fmt.Sprintf("%s:%s/%s",
				t.remote.Hostname,
				t.containerHomeDirectory,
				remoteName,
			)
		}
	}

	// Create the command.
	// We dont want to preserve user from host user, but use default container user.
	kubectlCommand := exec.Command(t.kubectlExecutable, "cp", "--no-preserve", localPath, containerPath)

	// Force it to run detached.
	kubectlCommand.SysProcAttr = process.DetachedProcessAttributes()

	// Create a copy of the current environment.
	environment := os.Environ()

	// Set Kubectl environment variables.
	environment = setKubectlVariables(environment, t.remote)

	// Set the environment for the command.
	kubectlCommand.Env = environment

	// Run the operation.
	if err := kubectlCommand.Run(); err != nil {
		return errors.Wrap(err, "unable to run Kubectl copy command")
	}

	// Success.
	return nil
}

// Command implements the Command method of agent.Transport.
func (t *transport) Command(command string) (*exec.Cmd, error) {
	// Ensure that the container has been probed.
	if err := t.probeContainer(); err != nil {
		return nil, errors.Wrap(err, "unable to probe container")
	}

	// Generate the command.
	return t.command(command), nil
}

// ClassifyError implements the ClassifyError method of agent.Transport.
func (t *transport) ClassifyError(processState *os.ProcessState, errorOutput string) (bool, bool, error) {
	// Ensure that the container has been probed.
	if err := t.probeContainer(); err != nil {
		return false, false, errors.Wrap(err, "unable to probe container")
	}

	// Kubectl alises cases of both "invalid command" (POSIX shell error 126) and
	// "command not found" (POSIX shell error 127) to an exit code of 126. It
	// even aliases the Windows container equivalents of these errors to 126.
	// Interestingly it even seems to have a 127 error code (see
	// https://github.com/moby/moby/pull/14012), though it's not returned when
	// the shell in the container generates a 127 exit code, so it's probably
	// just for its own internal commands.
	//
	// For POSIX containers, it's okay that it merges both of these errors,
	// since they lead to the same conclusion: the agent binary needs to be
	// (re-)installed. For Windows containers, it's a bit of a shame that both
	// error types get lumped together, because the "invalid command" error on
	// Windows is indicative of invoking a POSIX-style command inside a
	// cmd.exe-like environment, and detection of this error was one of the ways
	// that the agent package originally detected cmd.exe-like environments,
	// allowing for a reconnect attempt without a re-install attempt.
	// Fortunately the dialing code in the agent package will still attempt a
	// reconnect before a re-install if its platform hypothesis changes after
	// the first attempt, but I wish we could return more detailed information
	// to guide its decision.
	//
	// Anyway, the exit code we need to look out for with both POSIX and Windows
	// containers is 126, and since we know the remote platform already, we can
	// return that information without needing to resort to the error string.
	//
	// TODO: This is copied from Docker, verify if it is still true!
	//
	if !process.IsPOSIXShellInvalidCommand(processState) {
		return false, false, errors.New("unknown process exit error")
	}

	// Success.
	return true, t.containerIsWindows, nil
}
