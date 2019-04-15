package url

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	// KubectlURLPrefix is the lowercase version of the Kubectl URL prefix.
	KubectlURLPrefix = "kubectl://"

	// KubectlHostEnvironmentVariable is the name of the KUBECTL_HOST environment
	// variable.
	KubectlHostEnvironmentVariable = "KUBECTL_HOST"
	// KubectlTLSVerifyEnvironmentVariable is the name of the KUBECTL_TLS_VERIFY
	// environment variable.
	KubectlTLSVerifyEnvironmentVariable = "KUBECTL_TLS_VERIFY"
	// KubectlCertPathEnvironmentVariable is the name of the KUBECTL_CERT_PATH
	// environment variable.
	KubectlCertPathEnvironmentVariable = "KUBECTL_CERT_PATH"
)

// KubectlEnvironmentVariables is a list of Kubectl environment variables that
// should be locked in to the URL at parse time.
var KubectlEnvironmentVariables = []string{
	KubectlHostEnvironmentVariable,
	KubectlTLSVerifyEnvironmentVariable,
	KubectlCertPathEnvironmentVariable,
}

// isKubectlURL checks whether or not a URL is a Kubectl URL.
func isKubectlURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(raw), KubectlURLPrefix)
}

// parseKubectl parses a Kubectl URL.
func parseKubectl(raw string, alpha bool) (*URL, error) {
	// Strip off the prefix.
	raw = raw[len(KubectlURLPrefix):]

	// Split what remains into the pod and the path. Ideally we'd want to
	// be a bit more stringent here about what characters we accept in pod
	// names, potentially breaking early with an error if we see a "disallowed"
	// character, but we're better off just allowing Kubectl to reject pod
	// names that it doesn't like.
	var pod, path string
	for i, r := range raw {
		if r == '/' {
			pod = raw[:i]
			path = raw[i:]
			break
		}
	}
	if pod == "" {
		return nil, errors.New("empty pod name")
	} else if path == "" {
		return nil, errors.New("empty path")
	}

	// If the path starts with "/~", then we assume that it's supposed to be a
	// home-directory-relative path and remove the slash. At this point we
	// already know that the path starts with "/" since we retained that as part
	// of the path in the split operation above.
	if len(path) > 1 && path[1] == '~' {
		path = path[1:]
	}

	// If the path is of the form "/" + Windows path, then assume it's supposed
	// to be a Windows path. This is a heuristic, but a reasonable one. We do
	// this on all systems (not just on Windows as with SSH URLs) because users
	// can connect to Windows pods from non-Windows systems. At this point
	// we already know that the path starts with "/" since we retained that as
	// part of the path in the split operation above.
	if isWindowsPath(path[1:]) {
		path = path[1:]
	}

	// Loop over and record the values for the Kubectl environment variables that
	// we need to preserve. For the variables in question, Kubectl treats an
	// empty value the same as an unspecified value, so we always store
	// something for each variable, even if it's just an empty string to
	// indicate that its value was empty or unspecified.
	// TODO: I'm a little concerned that Kubectl may eventually add environment
	// variables where an empty value is not the same as an unspecified value,
	// but we'll cross that bridge when we come to it.
	environment := make(map[string]string, len(KubectlEnvironmentVariables))
	for _, variable := range KubectlEnvironmentVariables {
		value, _ := getEnvironmentVariable(variable, alpha)
		environment[variable] = value
	}

	// Success.
	return &URL{
		Protocol:    Protocol_Kubectl,
		Hostname:    pod,
		Path:        path,
		Environment: environment,
	}, nil
}
