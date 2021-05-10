package agent

import (
	"fmt"
	"os"
	"runtime"

	"github.com/google/uuid"

	"github.com/mutagen-io/mutagen/pkg/logging"
	"github.com/mutagen-io/mutagen/pkg/prompting"
)

// Install installs the current binary to the appropriate location for an agent
// binary with the current Mutagen version.
func Install() error {
	// Compute the destination.
	destination, err := installPath()
	if err != nil {
		return fmt.Errorf("unable to compute agent destination: %w", err)
	}

	// Compute the path to the current executable.
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("unable to determine executable path: %w", err)
	}

	// Relocate the current executable to the installation path.
	if err = os.Rename(executablePath, destination); err != nil {
		return fmt.Errorf("unable to relocate agent executable: %w", err)
	}

	// Success.
	return nil
}

// install attempts to probe an endpoint and install the appropriate agent
// binary over the specified transport.
func install(logger *logging.Logger, transport Transport, prompter string) error {
	// Detect the target platform.
	goos, goarch, posix, err := probe(transport, prompter)
	if err != nil {
		return fmt.Errorf("unable to probe remote platform: %w", err)
	}

	// Find the appropriate agent binary. Ensure that it's cleaned up when we're
	// done with it.
	if err := prompting.Message(prompter, "Extracting agent..."); err != nil {
		return fmt.Errorf("unable to message prompter: %w", err)
	}
	agentExecutable, err := executableForPlatform(goos, goarch)
	if err != nil {
		return fmt.Errorf("unable to get agent for platform: %w", err)
	}
	defer os.Remove(agentExecutable)

	// If we're not on Windows and our target system is not Windows, mark the
	// file as executable. This will save us an additional "chmod +x" operation
	// after copying the executable. If either side is Windows, then there's no
	// point in doing this, because either the bit won't be preserved (and hence
	// propagated by the copy operation) or it's not needed on the destination.
	if runtime.GOOS != "windows" && goos != "windows" {
		if err := os.Chmod(agentExecutable, 0700); err != nil {
			return fmt.Errorf("unable to make agent executable: %w", err)
		}
	}

	// Copy the agent to the remote. We use a unique identifier for the
	// temporary destination. For Windows remotes, we add a ".exe" suffix, which
	// will automatically make the file executable on the remote (POSIX systems
	// are handled separately below). For POSIX systems, we add a dot prefix to
	// hide the executable.
	if err := prompting.Message(prompter, "Copying agent..."); err != nil {
		return fmt.Errorf("unable to message prompter: %w", err)
	}
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("unable to generate UUID for agent copying: %w", err)
	}
	destination := BaseName + randomUUID.String()
	if goos == "windows" {
		destination += ".exe"
	}
	if posix {
		destination = "." + destination
	}
	if err = transport.Copy(agentExecutable, destination); err != nil {
		return fmt.Errorf("unable to copy agent binary: %w", err)
	}

	// For cases where we're copying from a Windows system to a POSIX remote,
	// invoke "chmod +x" to add executability back to the copied binary. This is
	// necessary under the specified circumstances because as soon as the agent
	// binary is extracted from the bundle, it will lose its executability bit
	// since Windows can't preserve this. This will also be applied to Windows
	// POSIX remotes, but a "chmod +x" there will just be a no-op.
	if runtime.GOOS == "windows" && posix {
		if err := prompting.Message(prompter, "Setting agent executability..."); err != nil {
			return fmt.Errorf("unable to message prompter: %w", err)
		}
		executabilityCommand := fmt.Sprintf("chmod +x %s", destination)
		if err := run(transport, executabilityCommand); err != nil {
			return fmt.Errorf("unable to set agent executability: %w", err)
		}
	}

	// Invoke the remote installation.
	if err := prompting.Message(prompter, "Installing agent..."); err != nil {
		return fmt.Errorf("unable to message prompter: %w", err)
	}
	var installCommand string
	if posix {
		installCommand = fmt.Sprintf("./%s %s", destination, ModeInstall)
	} else {
		installCommand = fmt.Sprintf("%s %s", destination, ModeInstall)
	}
	if err := run(transport, installCommand); err != nil {
		return fmt.Errorf("unable to invoke agent installation: %w", err)
	}

	// Success.
	return nil
}
