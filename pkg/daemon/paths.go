package daemon

import (
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

const (
	// lockName is the name of the daemon lock within the daemon subdirectory of
	// the Mutagen data directory.
	// TODO(LEGACY): Rename the lock to "lock" before v1.0.
	lockName = "daemon.lock"
	// endpointName is the name of the daemon IPC endpoint within the daemon
	// subdirectory of the Mutagen data directory.
	endpointName = "daemon.sock"
	// logName is the name of the daemon log file within the daemon subdirectory
	// of the Mutagen data directory.
	logName = "log"
	// tokenName is the name of the daemon token file within the daemon
	// subdirectory of the Mutagen data directory.
	tokenName = "token"
	// portName is the name of the daemon port file within the daemon
	// subdirectory of the Mutagen data directory.
	portName = "port"
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

// logPath computes the path to the daemon log, creating any intermediate
// directories as necessary.
func logPath() (string, error) {
	return subpath(logName)
}

// TokenPath computes the path to the daemon token file, creating any
// intermediate directories as necessary.
func TokenPath() (string, error) {
	return subpath(tokenName)
}

// PortPath computes the path to the daemon port file, creating any intermediate
// directories as necessary.
func PortPath() (string, error) {
	return subpath(portName)
}
