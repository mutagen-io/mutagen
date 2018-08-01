// +build !windows,!darwin

package daemon

import (
	"github.com/pkg/errors"
)

// RegistrationSupported indicates whether or not daemon registration is
// supported on this platform.
const RegistrationSupported = false

// Register performs automatic daemon startup registration.
func Register() error {
	return errors.New("daemon registration not supported on this platform")
}

// Unregister performs automatic daemon startup de-registration.
func Unregister() error {
	return errors.New("daemon deregistration not supported on this platform")
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
