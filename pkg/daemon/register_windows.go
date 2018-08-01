package daemon

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"golang.org/x/sys/windows/registry"
)

// RegistrationSupported indicates whether or not daemon registration is
// supported on this platform.
const RegistrationSupported = true

const (
	// rootKey is the registry root for daemon registration.
	rootKey = registry.CURRENT_USER
	// runPath is the path to the registry entries for automatic startup.
	runPath = "Software\\Microsoft\\Windows\\CurrentVersion\\Run"
	// runKeyName is the key used to register Mutagen for automatic startup.
	runKeyName = "Mutagen"
)

// Register performs automatic daemon startup registration.
func Register() error {
	// Attempt to open the relevant registry path and ensure it's cleaned up
	// when we're done.
	key, err := registry.OpenKey(rootKey, runPath, registry.SET_VALUE)
	if err != nil {
		return errors.Wrap(err, "unable to open registry path")
	}
	defer key.Close()

	// Compute the path to the current executable.
	executablePath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "unable to determine executable path")
	}

	// Compute the command to start the Mutagen daemon.
	command := fmt.Sprintf("\"%s\" daemon start", executablePath)

	// Attempt to register the daemon.
	if err := key.SetStringValue(runKeyName, command); err != nil {
		return errors.Wrap(err, "unable to set registry key")
	}

	// Success.
	return nil
}

// Unregister performs automatic daemon startup de-registration.
func Unregister() error {
	// Attempt to open the relevant registry path and ensure it's cleaned up
	// when we're done.
	key, err := registry.OpenKey(rootKey, runPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return errors.Wrap(err, "unable to open registry path")
	}
	defer key.Close()

	// Attempt to deregister the daemon.
	if err := key.DeleteValue(runKeyName); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "unable to remove registry key")
	}

	// Success.
	return nil
}

// RegisteredStart potentially handles daemon start operations if the daemon is
// registered for automatic start with the system. It returns false if the start
// operation was not handled and should be handled by the normal start command.
func RegisteredStart() (bool, error) {
	return false, nil
}

// RegisteredStop potentially handles stop start operations if the daemon is
// registered for automatic start with the system. It returns false if the stop
// operation was not handled and should be handled by the normal stop command.
func RegisteredStop() (bool, error) {
	return false, nil
}
