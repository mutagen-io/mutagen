package daemon

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/havoc-io/mutagen/pkg/filesystem"
)

const (
	// ipcEndpointName is the name of the daemon IPC endpoint. It resides within
	// the daemon subdirectory of the Mutagen directory.
	ipcEndpointName = "daemon.sock"
)

// subpath computes a subpath of the daemon subdirectory, creating the daemon
// subdirectory in the process.
func subpath(name string) (string, error) {
	// Compute the daemon root directory path and ensure it exists.
	daemonRoot, err := filesystem.Mutagen(true, filesystem.MutagenDaemonDirectoryName)
	if err != nil {
		return "", errors.Wrap(err, "unable to compute daemon directory")
	}

	// Compute the combined path.
	return filepath.Join(daemonRoot, name), nil
}

// IPCEndpointPath computes the path to the daemon IPC endpoint, creating any
// intermediate directories as necessary.
func IPCEndpointPath() (string, error) {
	return subpath(ipcEndpointName)
}
