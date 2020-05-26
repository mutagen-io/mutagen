package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/agent"
	"github.com/mutagen-io/mutagen/pkg/agent/transports/ssh"
	"github.com/mutagen-io/mutagen/pkg/docker"
	"github.com/mutagen-io/mutagen/pkg/process"
	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// windowsContainerNotification is a prompt about copying files into Windows
// containers, which requires stopping and re-starting the container.
const windowsContainerCopyNotification = `!!! ATTENTION !!!
In order to install its agent binary inside a Windows container, Mutagen will
need to stop and re-start the associated container. This is necessary because
Hyper-V doesn't support copying files into running containers.

Would you like to continue? (yes/no)? `

// transport implements the agent.Transport interface using Docker.
type transport struct {
	// container is the target container name.
	container string
	// user is the container user under which agents should be invoked.
	user string
	// environment is the collection of environment variables that need to be
	// set for the Docker executable.
	environment map[string]string
	// daemonConnectionFlags are the top-level flags used to control the daemon
	// connection. They are reconstituted from URL parameters.
	daemonConnectionFlags []string
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
	// containerUser is the name of the user inside the container. This will be
	// the same as the provided user, if any, but since that specification is
	// allowed to be empty (indicating a default user), we have to probe this
	// separately. It only applies if containerIsWindows is false.
	containerUser string
	// containerUserGroup is the name of the default group for the user inside
	// the container. It only applies if containerIsWindows is false.
	containerUserGroup string
	// containerProbeError tracks any error that arose when probing the
	// container.
	containerProbeError error
}

// NewTransport creates a new Docker transport using the specified parameters.
func NewTransport(container, user string, environment, parameters map[string]string, prompter string) (agent.Transport, error) {
	// Convert URL parameters to top-level daemon connection flags.
	daemonConnectionFlags, err := docker.LoadDaemonConnectionFlagsFromURLParameters(parameters)
	if err != nil {
		return nil, fmt.Errorf("unable to compute Docker daemon connection flags: %w", err)
	}

	// Success.
	return &transport{
		container:             container,
		user:                  user,
		environment:           environment,
		daemonConnectionFlags: daemonConnectionFlags.ToFlags(),
		prompter:              prompter,
	}, nil
}

// command is an underlying command generation function that allows
// specification of the working directory inside the container, as well as an
// override of the executing user. An empty user specification means to use the
// username specified in the remote URL, if any.
func (t *transport) command(command, workingDirectory, user string) (*exec.Cmd, error) {
	// Set up top-level command-line flags.
	var dockerArguments []string
	dockerArguments = append(dockerArguments, t.daemonConnectionFlags...)

	// Tell Docker that we want to execute a command in an interactive (i.e.
	// with standard input attached) fashion.
	dockerArguments = append(dockerArguments, "exec", "--interactive")

	// If specified, tell Docker which user should be used to execute commands
	// inside the container.
	if user != "" {
		dockerArguments = append(dockerArguments, "--user", user)
	} else if t.user != "" {
		dockerArguments = append(dockerArguments, "--user", t.user)
	}

	// If specified, tell Docker which directory should be used as the working
	// directory inside the container.
	if workingDirectory != "" {
		dockerArguments = append(dockerArguments, "--workdir", workingDirectory)
	}

	// Set the container name (this is stored as the Hostname field in the URL).
	dockerArguments = append(dockerArguments, t.container)

	// Lex the command that we want to run since Docker, unlike SSH, wants the
	// commands and arguments separately instead of as a single argument. All
	// agent.Transport interfaces only need to support commands that can be
	// lexed by splitting on spaces, so we don't need to pull in a more complex
	// shell lexing package here.
	dockerArguments = append(dockerArguments, strings.Split(command, " ")...)

	// Create the command.
	dockerCommand, err := docker.Command(context.Background(), dockerArguments...)
	if err != nil {
		return nil, err
	}

	// Force it to run detached.
	dockerCommand.SysProcAttr = process.DetachedProcessAttributes()

	// Create a copy of the current environment.
	environment := os.Environ()

	// Set Docker environment variables.
	environment = setDockerVariables(environment, t.environment)

	// Set SSH prompting environment variables. This is necessary to fully
	// support Docker's SSH protocol, which shells out to OpenSSH and thus may
	// require prompting.
	environment, err = ssh.SetPrompterVariables(environment, t.prompter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to set SSH prompting environment variables")
	}

	// Set the environment for the command.
	dockerCommand.Env = environment

	// Done.
	return dockerCommand, nil
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
	if command, err := t.command("env", "", ""); err != nil {
		return errors.Wrap(err, "unable to set up Docker invocation")
	} else if envBytes, err := command.Output(); err != nil {
		posixErr = err
	} else if !utf8.Valid(envBytes) {
		t.containerProbeError = errors.New("non-UTF-8 POSIX environment")
		return t.containerProbeError
	} else if env := string(envBytes); env == "" {
		t.containerProbeError = errors.New("empty POSIX environment")
		return t.containerProbeError
	} else if h, ok := findEnviromentVariable(env, "HOME"); !ok {
		t.containerProbeError = errors.New("unable to find home directory in POSIX environment")
		return t.containerProbeError
	} else if h == "" {
		t.containerProbeError = errors.New("empty POSIX home directory")
		return t.containerProbeError
	} else {
		home = h
	}

	// If we didn't find a POSIX home directory, attempt to a similar procedure
	// on Windows to identify the USERPROFILE environment variable.
	if home == "" {
		if command, err := t.command("cmd /c set", "", ""); err != nil {
			return errors.Wrap(err, "unable to set up Docker invocation")
		} else if envBytes, err := command.Output(); err != nil {
			windowsErr = err
		} else if !utf8.Valid(envBytes) {
			t.containerProbeError = errors.New("non-UTF-8 Windows environment")
			return t.containerProbeError
		} else if env := string(envBytes); env == "" {
			t.containerProbeError = errors.New("empty Windows environment")
			return t.containerProbeError
		} else if h, ok := findEnviromentVariable(env, "USERPROFILE"); !ok {
			t.containerProbeError = errors.New("unable to find home directory in Windows environment")
			return t.containerProbeError
		} else if h == "" {
			t.containerProbeError = errors.New("empty Windows home directory")
			return t.containerProbeError
		} else {
			home = h
			windows = true
		}
	}

	// If both probing mechanisms have failed, then create a combined error.
	if home == "" {
		t.containerProbeError = errors.Errorf(
			"container probing failed under POSIX hypothesis (%v) and Windows hypothesis (%v)",
			posixErr,
			windowsErr,
		)
		return t.containerProbeError
	}

	// At this point, home directory probing has succeeded. If we're using a
	// POSIX container, then attempt to extract the user's name and default
	// group so that we can set permissions on copied files. In theory, the
	// username should be the same as that passed in the URL, but we allow that
	// to be empty, which means the default user, usually but not necessarily
	// root. Since we need the explicit username to run our chown command, we
	// need to query it.
	var username, group string
	if !windows {
		// Query username.
		if command, err := t.command("id -un", "", ""); err != nil {
			return errors.Wrap(err, "unable to set up Docker invocation")
		} else if usernameBytes, err := command.Output(); err != nil {
			t.containerProbeError = errors.New("unable to probe POSIX username")
			return t.containerProbeError
		} else if !utf8.Valid(usernameBytes) {
			t.containerProbeError = errors.New("non-UTF-8 POSIX username")
			return t.containerProbeError
		} else if u := strings.TrimSpace(string(usernameBytes)); u == "" {
			t.containerProbeError = errors.New("empty POSIX username")
			return t.containerProbeError
		} else if t.user != "" && u != t.user {
			t.containerProbeError = errors.New("probed POSIX username does not match specified")
			return t.containerProbeError
		} else {
			username = u
		}

		// Query default group name.
		if command, err := t.command("id -gn", "", ""); err != nil {
			return errors.Wrap(err, "unable to set up Docker invocation")
		} else if groupBytes, err := command.Output(); err != nil {
			t.containerProbeError = errors.New("unable to probe POSIX group name")
			return t.containerProbeError
		} else if !utf8.Valid(groupBytes) {
			t.containerProbeError = errors.New("non-UTF-8 POSIX group name")
			return t.containerProbeError
		} else if g := strings.TrimSpace(string(groupBytes)); g == "" {
			t.containerProbeError = errors.New("empty POSIX group name")
			return t.containerProbeError
		} else {
			group = g
		}
	}

	// Store values.
	t.containerIsWindows = windows
	t.containerHomeDirectory = home
	t.containerUser = username
	t.containerUserGroup = group

	// Success.
	return nil
}

// changeContainerStatus stops or starts the container. It is required for
// copying files on Windows when using Hyper-V.
func (t *transport) changeContainerStatus(stop bool) error {
	// Set up top-level command-line flags.
	var dockerArguments []string
	dockerArguments = append(dockerArguments, t.daemonConnectionFlags...)

	// Set up the stop (or start) command.
	if stop {
		dockerArguments = append(dockerArguments, "stop", t.container)
	} else {
		dockerArguments = append(dockerArguments, "start", t.container)
	}

	// Create the command.
	dockerCommand, err := docker.Command(context.Background(), dockerArguments...)
	if err != nil {
		return errors.Wrap(err, "unable to set up Docker invocation")
	}

	// Force it to run detached.
	dockerCommand.SysProcAttr = process.DetachedProcessAttributes()

	// Create a copy of the current environment.
	environment := os.Environ()

	// Set Docker environment variables.
	environment = setDockerVariables(environment, t.environment)

	// Set SSH prompting environment variables. This is necessary to fully
	// support Docker's SSH protocol, which shells out to OpenSSH and thus may
	// require prompting.
	environment, err = ssh.SetPrompterVariables(environment, t.prompter)
	if err != nil {
		return errors.Wrap(err, "unable to set SSH prompting environment variables")
	}

	// Set the environment for the command.
	dockerCommand.Env = environment

	// Run the operation.
	return dockerCommand.Run()
}

// Copy implements the Copy method of agent.Transport.
func (t *transport) Copy(localPath, remoteName string) error {
	// Ensure that the container has been probed.
	if err := t.probeContainer(); err != nil {
		return errors.Wrap(err, "unable to probe container")
	}

	// If this is a Windows container, then we need to stop it from running
	// while we copy the agent. But first, we'll prompt the user to ensure that
	// they're okay with this.
	if t.containerIsWindows {
		if t.prompter == "" {
			return errors.New("no prompter for Docker copy behavior confirmation")
		}
		for {
			if response, err := prompting.Prompt(t.prompter, windowsContainerCopyNotification); err != nil {
				return errors.Wrap(err, "unable to prompt for Docker copy behavior confirmation")
			} else if response == "no" {
				return errors.New("user cancelled copy operation")
			} else if response == "yes" {
				break
			}
		}
		if err := t.changeContainerStatus(true); err != nil {
			return errors.Wrap(err, "unable to stop Docker container")
		}
	}

	// Compute the path inside the container. We don't bother trimming trailing
	// slashes from the home directory, because both Windows and POSIX will work
	// in their presence. The only case on Windows where \\ has special meaning
	// is with UNC paths, an in that case they only occur at the beginning of a
	// path, which they won't in this case since we've verified that the home
	// directory is non-empty.
	var containerPath string
	if t.containerIsWindows {
		containerPath = fmt.Sprintf("%s:%s\\%s",
			t.container,
			t.containerHomeDirectory,
			remoteName,
		)
	} else {
		containerPath = fmt.Sprintf("%s:%s/%s",
			t.container,
			t.containerHomeDirectory,
			remoteName,
		)
	}

	// Set up top-level command-line flags.
	var dockerArguments []string
	dockerArguments = append(dockerArguments, t.daemonConnectionFlags...)

	// Set up the copy command.
	dockerArguments = append(dockerArguments, "cp", localPath, containerPath)

	// Create the command.
	dockerCommand, err := docker.Command(context.Background(), dockerArguments...)
	if err != nil {
		return errors.Wrap(err, "unable to set up Docker invocation")
	}

	// Force it to run detached.
	dockerCommand.SysProcAttr = process.DetachedProcessAttributes()

	// Create a copy of the current environment.
	environment := os.Environ()

	// Set Docker environment variables.
	environment = setDockerVariables(environment, t.environment)

	// Set SSH prompting environment variables. This is necessary to fully
	// support Docker's SSH protocol, which shells out to OpenSSH and thus may
	// require prompting.
	environment, err = ssh.SetPrompterVariables(environment, t.prompter)
	if err != nil {
		return errors.Wrap(err, "unable to set SSH prompting environment variables")
	}

	// Set the environment for the command.
	dockerCommand.Env = environment

	// Run the operation.
	if err := dockerCommand.Run(); err != nil {
		return errors.Wrap(err, "unable to run Docker copy command")
	}

	// The default ownership of files copied into containers is a bit uncertain.
	//
	// For POSIX containers, ownership of the file is supposed to default to the
	// default container user and their associated default group (usually
	// root:root, which isn't always the user/group that we want), but
	// apparently that's not the case with Docker anymore due to a bug or
	// regression or just a behavioral change (see
	// https://github.com/moby/moby/issues/34096). In any case, the ownership
	// may be inappropriate for the file inside a POSIX container, so we
	// manually invoke chmod to set user/group ownership when dealing with this
	// container type. We always run this chmod command as root to ensure that
	// it succeeds.
	//
	// For Windows containers, there's no documented behavior. Through
	// experimentation, it seems like Docker just lets the file inherit the
	// permissions based on the path that it's copied into, which for home
	// directories is fine. If they change this in the future, we may need to
	// similarly probe the USERNAME environment variable and use icacls to set
	// ownership. It's a little unclear what user would be appropriate for
	// running this command, perhaps ContainerAdministrator if it is guaranteed
	// to exist, because most built-in NT accounts don't seem to exist in
	// containers.
	if !t.containerIsWindows {
		chownCommand := fmt.Sprintf(
			"chown %s:%s %s",
			t.containerUser,
			t.containerUserGroup,
			remoteName,
		)
		if command, err := t.command(chownCommand, t.containerHomeDirectory, "root"); err != nil {
			return errors.Wrap(err, "unable to set up Docker invocation")
		} else if err := command.Run(); err != nil {
			return errors.Wrap(err, "unable to set ownership of copied file")
		}
	}

	// If this is a Windows container, then we need to stop it from running
	// while we copy the agent.
	if t.containerIsWindows {
		if err := t.changeContainerStatus(false); err != nil {
			return errors.Wrap(err, "unable to start Docker container")
		}
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
	return t.command(command, t.containerHomeDirectory, "")
}

// ClassifyError implements the ClassifyError method of agent.Transport.
func (t *transport) ClassifyError(processState *os.ProcessState, errorOutput string) (bool, bool, error) {
	// Ensure that the container has been probed.
	if err := t.probeContainer(); err != nil {
		return false, false, errors.Wrap(err, "unable to probe container")
	}

	// Docker alises cases of both "invalid command" (POSIX shell error 126) and
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
	if !process.IsPOSIXShellInvalidCommand(processState) {
		return false, false, errors.New("unknown process exit error")
	}

	// Success.
	return true, t.containerIsWindows, nil
}
