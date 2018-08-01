package agent

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/mutagen"
	"github.com/havoc-io/mutagen/pkg/process"
)

const (
	// agentsDirectoryName is the subdirectory of the Mutagen directory in which
	// agents should be stored.
	agentsDirectoryName = "agents"
	// agentBaseName is the base name for agent executables (sans any
	// platform-specific suffix like ".exe").
	agentBaseName = "mutagen-agent"
)

// installPath computes and creates the parent directories of the path where the
// current executable should be installed if it is an agent binary with the
// current Mutagen version.
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
