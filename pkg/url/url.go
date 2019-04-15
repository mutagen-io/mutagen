package url

import (
	"github.com/pkg/errors"
)

// EnsureValid ensures that URL's invariants are respected.
func (u *URL) EnsureValid() error {
	// Ensure that the URL is non-nil.
	if u == nil {
		return errors.New("nil URL")
	}

	// Handle validation based on protocol.
	if u.Protocol == Protocol_Local {
		if u.Username != "" {
			return errors.New("local URL with non-empty username")
		} else if u.Hostname != "" {
			return errors.New("local URL with non-empty hostname")
		} else if u.Port != 0 {
			return errors.New("local URL with non-zero port")
		} else if u.Path == "" {
			return errors.New("local URL with empty path")
		} else if len(u.Environment) != 0 {
			return errors.New("local URL with environment variables")
		}
	} else if u.Protocol == Protocol_SSH {
		if u.Hostname == "" {
			return errors.New("SSH URL with empty hostname")
		} else if u.Path == "" {
			return errors.New("SSH URL with empty path")
		} else if len(u.Environment) != 0 {
			return errors.New("SSH URL with environment variables")
		}
	} else if u.Protocol == Protocol_Docker {
		// In the case of Docker, we intentionally avoid validating environment
		// variables since the values used could change over time. Since we
		// default to empty values for unspecified environment variables, this
		// works out fine, at least so long as Docker continues to treat empty
		// environment variables the same as unspecified ones.
		if u.Hostname == "" {
			return errors.New("Docker URL with empty container identifier")
		} else if u.Port != 0 {
			return errors.New("Docker URL with non-zero port")
		} else if u.Path == "" {
			return errors.New("Docker URL with empty path")
		} else if !(u.Path[0] == '/' || u.Path[0] == '~' || isWindowsPath(u.Path)) {
			return errors.New("Docker URL with incorrect first path character")
		}
	} else if u.Protocol == Protocol_Kubectl {
		// In the case of Kubectl, we intentionally avoid validating environment
		// variables since the values used could change over time. Since we
		// default to empty values for unspecified environment variables, this
		// works out fine, at least so long as Kubectl continues to treat empty
		// environment variables the same as unspecified ones.
		if u.Hostname == "" {
			return errors.New("Kubectl URL with empty container identifier")
		} else if u.Port != 0 {
			return errors.New("Kubectl URL with non-zero port")
		} else if u.Path == "" {
			return errors.New("Kubectl URL with empty path")
		} else if !(u.Path[0] == '/' || u.Path[0] == '~' || isWindowsPath(u.Path)) {
			return errors.New("Kubectl URL with incorrect first path character")
		}
	} else {
		return errors.New("unknown or unsupported protocol")
	}

	// Success.
	return nil
}
