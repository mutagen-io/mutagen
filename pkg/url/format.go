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
	} else if u.Protocol == Protocol_Docker {
		return u.formatDocker(environmentPrefix)
	} else if u.Protocol == Protocol_Kubectl {
		return u.formatKubectl(environmentPrefix)
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

// invalidDockerURLFormat is the value returned by formatDocker when a URL is
// provided that breaks invariants.
const invalidDockerURLFormat = "<invalid-docker-url>"

// formatDocker formats a Docker URL.
func (u *URL) formatDocker(environmentPrefix string) string {
	// Start with the container name.
	result := u.Hostname

	// Append the path. If this is a home-directory-relative path or a Windows
	// path, then we need to prepend a slash.
	if u.Path == "" {
		return invalidDockerURLFormat
	} else if u.Path[0] == '/' {
		result += u.Path
	} else if u.Path[0] == '~' || isWindowsPath(u.Path) {
		result += fmt.Sprintf("/%s", u.Path)
	} else {
		return invalidDockerURLFormat
	}

	// Add username if present.
	if u.Username != "" {
		result = fmt.Sprintf("%s@%s", u.Username, result)
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

// invalidKubectlURLFormat is the value returned by formatKubectl when a URL is
// provided that breaks invariants.
const invalidKubectlURLFormat = "<invalid-kubectl-url>"

// formatKubectl formats a Kubectl URL.
func (u *URL) formatKubectl(environmentPrefix string) string {
	// Start with the container name.
	result := u.Hostname

	// Append the path. If this is a home-directory-relative path or a Windows
	// path, then we need to prepend a slash.
	if u.Path == "" {
		return invalidKubectlURLFormat
	} else if u.Path[0] == '/' {
		result += u.Path
	} else if u.Path[0] == '~' || isWindowsPath(u.Path) {
		result += fmt.Sprintf("/%s", u.Path)
	} else {
		return invalidKubectlURLFormat
	}

	// Add the scheme.
	result = KubectlURLPrefix + result

	// Add environment variable information if requested.
	if environmentPrefix != "" {
		for _, variable := range KubectlEnvironmentVariables {
			result += fmt.Sprintf("%s%s=%s", environmentPrefix, variable, u.Environment[variable])
		}
	}

	// Done.
	return result
}
