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
	// sourceSpecificEnvironmentVariablePrefix is the prefix to use when
	// checking for source-specific environment variables.
	sourceSpecificEnvironmentVariablePrefix = "MUTAGEN_SOURCE_"
	// destinationSpecificEnvironmentVariablePrefix is the prefix to use when
	// checking for destination-specific environment variables.
	destinationSpecificEnvironmentVariablePrefix = "MUTAGEN_DESTINATION_"
)

// lookupEnv is the environment variable lookup function to use. It is a
// variable so that it can be swapped out during testing.
var lookupEnv = os.LookupEnv

// getEnvironmentVariable returns the value for the specified environment
// variable, as well as whether or not it was found. Endpoint-specific variables
// take precedence over non-specific variables.
func getEnvironmentVariable(name string, kind Kind, first bool) (string, bool) {
	// Validate the variable name.
	if name == "" {
		return "", false
	}

	// Check for an endpoint-specific variant.
	var endpointSpecificName string
	if kind == Kind_Synchronization {
		if first {
			endpointSpecificName = alphaSpecificEnvironmentVariablePrefix + name
		} else {
			endpointSpecificName = betaSpecificEnvironmentVariablePrefix + name
		}
	} else if kind == Kind_Forwarding {
		if first {
			endpointSpecificName = sourceSpecificEnvironmentVariablePrefix + name
		} else {
			endpointSpecificName = destinationSpecificEnvironmentVariablePrefix + name
		}
	} else {
		panic("unhandled URL kind")
	}
	if value, ok := lookupEnv(endpointSpecificName); ok {
		return value, true
	}

	// Check for the general variant.
	return lookupEnv(name)
}
