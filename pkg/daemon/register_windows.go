package daemon

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"golang.org/x/sys/windows/registry"

	"github.com/havoc-io/mutagen/pkg/process"
)

const RegistrationSupported = true

const (
	rootKey    = registry.CURRENT_USER
	runPath    = "Software\\Microsoft\\Windows\\CurrentVersion\\Run"
	runKeyName = "Mutagen"
)

func Register() error {
	// Attempt to open the relevant registry path and ensure it's cleaned up
	// when we're done.
	key, err := registry.OpenKey(rootKey, runPath, registry.SET_VALUE)
	if err != nil {
		return errors.Wrap(err, "unable to open registry path")
	}
	defer key.Close()

	// Compute the command to start the Mutagen daemon.
	command := fmt.Sprintf("\"%s\" daemon start", process.Current.ExecutablePath)

	// Attempt to register the daemon.
	if err := key.SetStringValue(runKeyName, command); err != nil {
		return errors.Wrap(err, "unable to set registry key")
	}

	// Success.
	return nil
}

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

func StartStopAllowed() (bool, error) {
	return true, nil
}
