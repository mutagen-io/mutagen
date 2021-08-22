package daemon

import (
	"fmt"
	"path/filepath"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

const (
	// lockName is the name of the daemon lock. It resides within the daemon
	// subdirectory of the Mutagen directory.
	lockName = "daemon.lock"
	// endpointName is the name of the daemon IPC endpoint. It resides within
	// the daemon subdirectory of the Mutagen directory.
	endpointName = "daemon.sock"
)

// subpath computes a subpath of the daemon subdirectory, creating the daemon
// subdirectory in the process.
func subpath(name string) (string, error) {
	// Compute the daemon root directory path and ensure it exists.
	daemonRoot, err := filesystem.Mutagen(true, filesystem.MutagenDaemonDirectoryName)
	if err != nil {
		return "", fmt.Errorf("unable to compute daemon directory: %w", err)
	}

	// Compute the combined path.
	return filepath.Join(daemonRoot, name), nil
}

// lockPath computes the path to the daemon lock, creating any intermediate
// directories as necessary.
func lockPath() (string, error) {
	return subpath(lockName)
}

// EndpointPath computes the path to the daemon IPC endpoint, creating any
// intermediate directories as necessary.
func EndpointPath() (string, error) {
	return subpath(endpointName)
}
