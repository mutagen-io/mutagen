package daemon

import (
	"github.com/pkg/errors"
)

const RegistrationSupported = false

func Register() error {
	return errors.New("daemon registration not yet implemented on this platform")
}

func Unregister() error {
	return errors.New("daemon deregistration not yet implemented on this platform")
}

func StartStopAllowed() (bool, error) {
	return true, nil
}
