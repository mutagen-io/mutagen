package url

import (
	"fmt"
)

// Format formats a URL into a human-readable (and reparsable) format.
func (u *URL) Format(environmentPrefix string) string {
	if u.Protocol == Protocol_Local {
		return u.formatLocal()
	} else if u.Protocol == Protocol_SSH {
		return u.formatSSH()
	} else if u.Protocol == Protocol_Tunnel {
		return u.formatTunnel()
	} else if u.Protocol == Protocol_Docker {
		return u.formatDocker(environmentPrefix)
	}
	panic("unknown URL protocol")
}

// formatLocal formats a local URL.
func (u *URL) formatLocal() string {
	return u.Path
}

// formatSSH formats an SSH URL into an SCP-style URL.
func (u *URL) formatSSH() string {
	// Create the base result.
	result := u.Host

	// Add username if present.
	if u.User != "" {
		result = fmt.Sprintf("%s@%s", u.User, result)
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

// invalidTunnelURLFormat is the value returned by formatTunnel when a URL is
// provided that breaks invariants.
const invalidTunnelURLFormat = "<invalid-tunnel-url>"

// formatTunnel formats a tunnel URL.
func (u *URL) formatTunnel() string {
	// Start with the tunnel identifier/name.
	result := u.Host

	// Ensure that a username is not present
	if u.User != "" {
		return invalidTunnelURLFormat
	}

	// Append the path in a manner that depends on the URL kind.
	if u.Kind == Kind_Synchronization {
		// If this is a home-directory-relative path or a Windows path, then we
		// need to prepend a slash.
		if u.Path == "" {
			return invalidTunnelURLFormat
		} else if u.Path[0] == '/' {
			result += u.Path
		} else if u.Path[0] == '~' || isWindowsPath(u.Path) {
			result += fmt.Sprintf("/%s", u.Path)
		} else {
			return invalidTunnelURLFormat
		}
	} else if u.Kind == Kind_Forwarding {
		result += fmt.Sprintf(":%s", u.Path)
	} else {
		panic("unhandled URL kind")
	}

	// Add the scheme.
	result = tunnelURLPrefix + result

	// Done.
	return result
}

// invalidDockerURLFormat is the value returned by formatDocker when a URL is
// provided that breaks invariants.
const invalidDockerURLFormat = "<invalid-docker-url>"

// formatDocker formats a Docker URL.
func (u *URL) formatDocker(environmentPrefix string) string {
	// Start with the container name.
	result := u.Host

	// Add username if present.
	if u.User != "" {
		result = fmt.Sprintf("%s@%s", u.User, result)
	}

	// Append the path in a manner that depends on the URL kind.
	if u.Kind == Kind_Synchronization {
		// If this is a home-directory-relative path or a Windows path, then we
		// need to prepend a slash.
		if u.Path == "" {
			return invalidDockerURLFormat
		} else if u.Path[0] == '/' {
			result += u.Path
		} else if u.Path[0] == '~' || isWindowsPath(u.Path) {
			result += fmt.Sprintf("/%s", u.Path)
		} else {
			return invalidDockerURLFormat
		}
	} else if u.Kind == Kind_Forwarding {
		result += fmt.Sprintf(":%s", u.Path)
	} else {
		panic("unhandled URL kind")
	}

	// Add the scheme.
	result = dockerURLPrefix + result

	// Add environment variable information if requested.
	if environmentPrefix != "" {
		for _, variable := range DockerEnvironmentVariables {
			result += fmt.Sprintf("%s%s=%s", environmentPrefix, variable, u.Environment[variable])
		}
	}

	// Done.
	return result
}
