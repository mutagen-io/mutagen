package url

import (
	"testing"
)

const (
	// alphaSpecificDockerHostEnvironmentVariable is the name of the
	// alpha-specific DOCKER_HOST environment variable.
	alphaSpecificDockerHostEnvironmentVariable = "MUTAGEN_ALPHA_DOCKER_HOST"
	// alphaSpecificDockerHost is the alpha-specific value for the DOCKER_HOST
	// environment variable.
	alphaSpecificDockerHost = "unix:///alpha/docker.sock"
	// betaSpecificDockerTLSVerifyEnvironmentVariable is the name of the
	// beta-specific DOCKER_TLS_VERIFY environment variable.
	betaSpecificDockerTLSVerifyEnvironmentVariable = "MUTAGEN_BETA_DOCKER_TLS_VERIFY"
	// betaSpecificDockerTLSVerify is the beta-specific value for the
	// DOCKER_TLS_VERIFY environment variable.
	betaSpecificDockerTLSVerify = "true"
	// sourceSpecificDockerHostEnvironmentVariable is the name of the
	// source-specific DOCKER_HOST environment variable.
	sourceSpecificDockerHostEnvironmentVariable = "MUTAGEN_SOURCE_DOCKER_HOST"
	// sourceSpecificDockerHost is the source-specific value for the DOCKER_HOST
	// environment variable.
	sourceSpecificDockerHost = "unix:///source/docker.sock"
	// destinationSpecificDockerTLSVerifyEnvironmentVariable is the name of the
	// destination-specific DOCKER_TLS_VERIFY environment variable.
	destinationSpecificDockerTLSVerifyEnvironmentVariable = "MUTAGEN_DESTINATION_DOCKER_TLS_VERIFY"
	// destinationSpecificDockerTLSVerify is the destination-specific value for
	// the DOCKER_TLS_VERIFY environment variable.
	destinationSpecificDockerTLSVerify = "false"
	// defaultDockerHost is the non-endpoint-specific value for the DOCKER_HOST
	// environment variable.
	defaultDockerHost = "unix:///default/docker.sock"
	// defaultDockerTLSVerify is the non-endpoint-specific value for the
	// DOCKER_TLS_VERIFY environment variable.
	defaultDockerTLSVerify = "sure!"
)

// mockEnvironment is a mock environment setup for use in testing.
var mockEnvironment = map[string]string{
	DockerHostEnvironmentVariable:                         defaultDockerHost,
	DockerTLSVerifyEnvironmentVariable:                    defaultDockerTLSVerify,
	alphaSpecificDockerHostEnvironmentVariable:            alphaSpecificDockerHost,
	betaSpecificDockerTLSVerifyEnvironmentVariable:        betaSpecificDockerTLSVerify,
	sourceSpecificDockerHostEnvironmentVariable:           sourceSpecificDockerHost,
	destinationSpecificDockerTLSVerifyEnvironmentVariable: destinationSpecificDockerTLSVerify,
}

// mockLookupEnv is a mock implementation of the os.LookupEnv function.
func mockLookupEnv(name string) (string, bool) {
	value, ok := mockEnvironment[name]
	return value, ok
}

func init() {
	// Replace the lookupEnv function with one that uses a mock environment.
	lookupEnv = mockLookupEnv
}

func TestAlphaLookupAlphaSpecificExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerHostEnvironmentVariable, Kind_Synchronization, true); !ok {
		t.Fatal("unable to find alpha-specific value")
	} else if value != alphaSpecificDockerHost {
		t.Fatal("alpha-specific value does not match expected")
	}
}

func TestAlphaLookupOnlyDefaultExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerTLSVerifyEnvironmentVariable, Kind_Synchronization, true); !ok {
		t.Fatal("unable to find non-endpoint-specific value for alpha")
	} else if value != defaultDockerTLSVerify {
		t.Fatal("non-endpoint-specific value does not match expected")
	}
}

func TestAlphaLookupNeitherExists(t *testing.T) {
	if _, ok := getEnvironmentVariable(DockerCertPathEnvironmentVariable, Kind_Synchronization, true); ok {
		t.Fatal("able to find unset environment variable")
	}
}

func TestBetaLookupBetaSpecificExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerTLSVerifyEnvironmentVariable, Kind_Synchronization, false); !ok {
		t.Fatal("unable to find beta-specific value")
	} else if value != betaSpecificDockerTLSVerify {
		t.Fatal("beta-specific value does not match expected")
	}
}

func TestBetaLookupOnlyDefaultExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerHostEnvironmentVariable, Kind_Synchronization, false); !ok {
		t.Fatal("unable to find non-endpoint-specific value for alpha")
	} else if value != defaultDockerHost {
		t.Fatal("non-endpoint-specific value does not match expected")
	}
}

func TestBetaLookupNeitherExists(t *testing.T) {
	if _, ok := getEnvironmentVariable(DockerCertPathEnvironmentVariable, Kind_Synchronization, true); ok {
		t.Fatal("able to find unset environment variable")
	}
}

func TestSourceLookupSourceSpecificExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerHostEnvironmentVariable, Kind_Forwarding, true); !ok {
		t.Fatal("unable to find source-specific value")
	} else if value != sourceSpecificDockerHost {
		t.Fatal("source-specific value does not match expected")
	}
}

func TestSourceLookupOnlyDefaultExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerTLSVerifyEnvironmentVariable, Kind_Forwarding, true); !ok {
		t.Fatal("unable to find non-endpoint-specific value for source")
	} else if value != defaultDockerTLSVerify {
		t.Fatal("non-endpoint-specific value does not match expected")
	}
}

func TestSourceLookupNeitherExists(t *testing.T) {
	if _, ok := getEnvironmentVariable(DockerCertPathEnvironmentVariable, Kind_Forwarding, true); ok {
		t.Fatal("able to find unset environment variable")
	}
}

func TestDestinationLookupDestinationSpecificExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerTLSVerifyEnvironmentVariable, Kind_Forwarding, false); !ok {
		t.Fatal("unable to find destination-specific value")
	} else if value != destinationSpecificDockerTLSVerify {
		t.Fatal("destination-specific value does not match expected")
	}
}

func TestDestinationLookupOnlyDefaultExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerHostEnvironmentVariable, Kind_Forwarding, false); !ok {
		t.Fatal("unable to find non-endpoint-specific value for source")
	} else if value != defaultDockerHost {
		t.Fatal("non-endpoint-specific value does not match expected")
	}
}

func TestDestinationLookupNeitherExists(t *testing.T) {
	if _, ok := getEnvironmentVariable(DockerCertPathEnvironmentVariable, Kind_Forwarding, true); ok {
		t.Fatal("able to find unset environment variable")
	}
}
