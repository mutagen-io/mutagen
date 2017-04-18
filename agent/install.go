package agent

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen"
	"github.com/havoc-io/mutagen/filesystem"
	"github.com/havoc-io/mutagen/process"
)

const (
	agentsDirectoryName = "agents"
	agentBaseName       = "mutagen-agent"
)

func installPath() (string, error) {
	// Compute (and create) the path to the agent parent directory.
	parent, err := filesystem.Mutagen(true, agentsDirectoryName, mutagen.Version)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute parent directory")
	}

	// Compute the target executable name.
	executableName := process.ExecutableName(agentBaseName, runtime.GOOS)

	// Compute the installation path.
	return filepath.Join(parent, executableName), nil
}

func Install() error {
	// Compute the destination.
	destination, err := installPath()
	if err != nil {
		return errors.Wrap(err, "unable to compute agent destination")
	}

	// Relocate the current executable to the installation path.
	if err = os.Rename(process.Current.ExecutablePath, destination); err != nil {
		return errors.Wrap(err, "unable to relocate agent executable")
	}

	// Success.
	return nil
}
