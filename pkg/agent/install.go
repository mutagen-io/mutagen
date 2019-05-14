package agent

import (
	"fmt"
	"os"
	"runtime"

	"github.com/pkg/errors"

	"github.com/google/uuid"

	"github.com/havoc-io/mutagen/pkg/prompt"
)

// Install installs the current binary to the appropriate location for an agent
// binary with the current Mutagen version.
func Install() error {
	// Compute the destination.
	destination, err := installPath()
	if err != nil {
		return errors.Wrap(err, "unable to compute agent destination")
	}

	// Compute the path to the current executable.
	executablePath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "unable to determine executable path")
	}

	// Relocate the current executable to the installation path.
	if err = os.Rename(executablePath, destination); err != nil {
		return errors.Wrap(err, "unable to relocate agent executable")
	}

	// Success.
	return nil
}

// install attempts to probe an endpoint and install the appropriate agent
// binary over the specified transport.
func install(transport Transport, prompter string) error {
	// Detect the target platform.
	goos, goarch, posix, err := probe(transport, prompter)
	if err != nil {
		return errors.Wrap(err, "unable to probe remote platform")
	}

	// Find the appropriate agent binary. Ensure that it's cleaned up when we're
	// done with it.
	if err := prompt.Message(prompter, "Extracting agent..."); err != nil {
		return errors.Wrap(err, "unable to message prompter")
	}
	agentExecutable, err := executableForPlatform(goos, goarch)
	if err != nil {
		return errors.Wrap(err, "unable to get agent for platform")
	}
	defer os.Remove(agentExecutable)

	// Copy the agent to the remote. We use a unique identifier for the
	// temporary destination. For Windows remotes, we add a ".exe" suffix, which
	// will automatically make the file executable on the remote (POSIX systems
	// are handled separately below). For POSIX systems, we add a dot prefix to
	// hide the executable.
	if err := prompt.Message(prompter, "Copying agent..."); err != nil {
		return errors.Wrap(err, "unable to message prompter")
	}
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return errors.Wrap(err, "unable to generate UUID for agent copying")
	}
	destination := BaseName + randomUUID.String()
	if goos == "windows" {
		destination += ".exe"
	}
	if posix {
		destination = "." + destination
	}
	if err = transport.Copy(agentExecutable, destination); err != nil {
		return errors.Wrap(err, "unable to copy agent binary")
	}

	// For cases where we're copying from a Windows system to a POSIX remote,
	// invoke "chmod +x" to add executability back to the copied binary. This is
	// necessary under the specified circumstances because as soon as the agent
	// binary is extracted from the bundle, it will lose its executability bit
	// since Windows can't preserve this. This will also be applied to Windows
	// POSIX remotes, but a "chmod +x" there will just be a no-op.
	if runtime.GOOS == "windows" && posix {
		if err := prompt.Message(prompter, "Setting agent executability..."); err != nil {
			return errors.Wrap(err, "unable to message prompter")
		}
		executabilityCommand := fmt.Sprintf("chmod +x %s", destination)
		if err := run(transport, executabilityCommand); err != nil {
			return errors.Wrap(err, "unable to set agent executability")
		}
	}

	// Invoke the remote installation.
	if err := prompt.Message(prompter, "Installing agent..."); err != nil {
		return errors.Wrap(err, "unable to message prompter")
	}
	var installCommand string
	if posix {
		installCommand = fmt.Sprintf("./%s %s", destination, ModeInstall)
	} else {
		installCommand = fmt.Sprintf("%s %s", destination, ModeInstall)
	}
	if err := run(transport, installCommand); err != nil {
		return errors.Wrap(err, "unable to invoke agent installation")
	}

	// Success.
	return nil
}
