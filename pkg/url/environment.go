package url

import (
	"os"
)

const (
	// alphaSpecificEnvironmentVariablePrefix is the prefix to use when checking
	// for alpha-specific environment variables.
	alphaSpecificEnvironmentVariablePrefix = "MUTAGEN_ALPHA_"
	// betaSpecificEnvironmentVariablePrefix is the prefix to use when checking
	// for beta-specific environment variables.
	betaSpecificEnvironmentVariablePrefix = "MUTAGEN_BETA_"
)

// lookupEnv is the environment variable lookup function to use. It is a
// variable so that it can be swapped out during testing.
var lookupEnv = os.LookupEnv

// getEnvironmentVariable returns the value for the specified environment
// variable, as well as whether or not it was found. Endpoint-specific variables
// take precedence over non-specific variables.
func getEnvironmentVariable(name string, alpha bool) (string, bool) {
	// Validate the variable name.
	if name == "" {
		return "", false
	}

	// Check for an endpoint-specific variant.
	var endpointSpecificName string
	if alpha {
		endpointSpecificName = alphaSpecificEnvironmentVariablePrefix + name
	} else {
		endpointSpecificName = betaSpecificEnvironmentVariablePrefix + name
	}
	if value, ok := lookupEnv(endpointSpecificName); ok {
		return value, true
	}

	// Check for the general variant.
	return lookupEnv(name)
}
