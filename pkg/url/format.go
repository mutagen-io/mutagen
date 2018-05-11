package url

import (
	"fmt"
)

func (u *URL) Format() string {
	if u.Protocol == Protocol_Local {
		return u.formatLocal()
	} else if u.Protocol == Protocol_SSH {
		return u.formatSSH()
	}
	panic("unknown URL protocol")
}

func (u *URL) formatLocal() string {
	return u.Path
}

func (u *URL) formatSSH() string {
	// Create the base result.
	result := u.Hostname

	// Add username if present.
	if u.Username != "" {
		result = fmt.Sprintf("%s@%s", u.Username, result)
	}

	// Add port if present.
	if u.Port != 0 {
		result = fmt.Sprintf("%s:%d", result, u.Port)
	}

	// Add path.
	result = fmt.Sprintf("%s:%s", result, u.Path)

	// Done.
	return result
}
