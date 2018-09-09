package agent

import (
	"os"

	"github.com/pkg/errors"
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
