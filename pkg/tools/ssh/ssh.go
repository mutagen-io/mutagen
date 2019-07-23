package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/process"
)

// CompressionArgument returns a flag that can be passed to scp or ssh to enable
// compression. Note that while SSH does have a CompressionLevel configuration
// option, this only applies to SSHv1. SSHv2 defaults to a DEFLATE level of 6,
// which is what we want anyway.
func CompressionArgument() string {
	return "-C"
}

// TimeoutArgument returns a option flag that can be passed to scp or ssh to
// limit connection time (though not transfer time or process lifetime). The
// provided timeout is in seconds. It must be > 0, otherwise this function will
// panic.
func TimeoutArgument(timeout int) string {
	// Validate the timeout.
	if timeout < 1 {
		panic("invalid timeout value")
	}

	// Format the argument.
	return fmt.Sprintf("-oConnectTimeout=%d", timeout)
}

// sshCommandPath returns the full path to use for invoking ssh. It will use the
// MUTAGEN_SSH_PATH environment variable if provided, otherwise falling back to
// a platform-specific implementation.
func sshCommandPath() (string, error) {
	// If MUTAGEN_SSH_PATH is specified, then use it to perform the lookup.
	if searchPath := os.Getenv("MUTAGEN_SSH_PATH"); searchPath != "" {
		return process.FindCommand("ssh", []string{searchPath})
	}

	// Otherwise fall back to the platform-specific implementation.
	return sshCommandPathForPlatform()
}

// SSHCommand prepares (but does not start) an SSH command with the specified
// arguments and scoped to lifetime of the provided context.
func SSHCommand(context context.Context, args ...string) (*exec.Cmd, error) {
	// Identify the command name or path.
	nameOrPath, err := sshCommandPath()
	if err != nil {
		return nil, errors.Wrap(err, "unable to identify 'ssh' command")
	}

	// Create the command.
	return exec.CommandContext(context, nameOrPath, args...), nil
}

// scpCommandPath returns the full path to use for invoking scp. It will use the
// MUTAGEN_SSH_PATH environment variable if provided, otherwise falling back to
// a platform-specific implementation.
func scpCommandPath() (string, error) {
	// If MUTAGEN_SSH_PATH is specified, then use it to perform the lookup.
	if searchPath := os.Getenv("MUTAGEN_SSH_PATH"); searchPath != "" {
		return process.FindCommand("scp", []string{searchPath})
	}

	// Otherwise fall back to the platform-specific implementation.
	return scpCommandPathForPlatform()
}

// SCPCommand prepares (but does not start) an SCP command with the specified
// arguments and scoped to lifetime of the provided context.
func SCPCommand(context context.Context, args ...string) (*exec.Cmd, error) {
	// Identify the command name or path.
	nameOrPath, err := scpCommandPath()
	if err != nil {
		return nil, errors.Wrap(err, "unable to identify 'scp' command")
	}

	// Create the command.
	return exec.CommandContext(context, nameOrPath, args...), nil
}
