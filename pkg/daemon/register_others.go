// +build !windows,!darwin,!linux

package daemon

import (
	"github.com/pkg/errors"
)

const RegistrationSupported = false

func Register() error {
	return errors.New("daemon registration not supported on this platform")
}

func Unregister() error {
	return errors.New("daemon deregistration not supported on this platform")
}

func StartStopAllowed() (bool, error) {
	return true, nil
}

func RegisteredStart() (bool, error) {
	return false, nil
}

func RegisteredStop() (bool, error) {
	return false, nil
}
