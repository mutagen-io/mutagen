package url

import (
	"testing"
)

const (
	// defaultDockerHost is the non-endpoint-specific value for the DOCKER_HOST
	// environment variable.
	defaultDockerHost = "unix:///default/docker.sock"
	// defaultDockerTLSVerify is the non-endpoint-specific value for the
	// DOCKER_TLS_VERIFY environment variable.
	defaultDockerTLSVerify = "sure!"
	// alphaSpecificDockerHost is the alpha-specific value for the DOCKER_HOST
	// environment variable.
	alphaSpecificDockerHost = "unix:///alpha/docker.sock"
	// betaSpecificDockerTLSVerify is the beta-specific value for the
	// DOCKER_TLS_VERIFY environment variable.
	betaSpecificDockerTLSVerify = "true"
	// sourceSpecificDockerContext is the source-specific value for the
	// DOCKER_CONTEXT environment variable.
	sourceSpecificDockerContext = "some-context"
	// destinationSpecificDockerTLSVerify is the destination-specific value for
	// the DOCKER_TLS_VERIFY environment variable.
	destinationSpecificDockerTLSVerify = "false"
)

// mockEnvironment is a mock environment setup for use in testing.
var mockEnvironment = map[string]string{
	"DOCKER_HOST":                           defaultDockerHost,
	"DOCKER_TLS_VERIFY":                     defaultDockerTLSVerify,
	"MUTAGEN_ALPHA_DOCKER_HOST":             alphaSpecificDockerHost,
	"MUTAGEN_BETA_DOCKER_TLS_VERIFY":        betaSpecificDockerTLSVerify,
	"MUTAGEN_SOURCE_DOCKER_CONTEXT":         sourceSpecificDockerContext,
	"MUTAGEN_DESTINATION_DOCKER_TLS_VERIFY": destinationSpecificDockerTLSVerify,
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
	if value, ok := getEnvironmentVariable("DOCKER_HOST", Kind_Synchronization, true); !ok {
		t.Fatal("unable to find alpha-specific value")
	} else if value != alphaSpecificDockerHost {
		t.Fatal("alpha-specific value does not match expected")
	}
}

func TestAlphaLookupOnlyDefaultExists(t *testing.T) {
	if value, ok := getEnvironmentVariable("DOCKER_TLS_VERIFY", Kind_Synchronization, true); !ok {
		t.Fatal("unable to find non-endpoint-specific value for alpha")
	} else if value != defaultDockerTLSVerify {
		t.Fatal("non-endpoint-specific value does not match expected")
	}
}

func TestAlphaLookupNeitherExists(t *testing.T) {
	if _, ok := getEnvironmentVariable("DOCKER_CERT_PATH", Kind_Synchronization, true); ok {
		t.Fatal("able to find unset environment variable")
	}
}

func TestBetaLookupBetaSpecificExists(t *testing.T) {
	if value, ok := getEnvironmentVariable("DOCKER_TLS_VERIFY", Kind_Synchronization, false); !ok {
		t.Fatal("unable to find beta-specific value")
	} else if value != betaSpecificDockerTLSVerify {
		t.Fatal("beta-specific value does not match expected")
	}
}

func TestBetaLookupOnlyDefaultExists(t *testing.T) {
	if value, ok := getEnvironmentVariable("DOCKER_HOST", Kind_Synchronization, false); !ok {
		t.Fatal("unable to find non-endpoint-specific value for alpha")
	} else if value != defaultDockerHost {
		t.Fatal("non-endpoint-specific value does not match expected")
	}
}

func TestBetaLookupNeitherExists(t *testing.T) {
	if _, ok := getEnvironmentVariable("DOCKER_CERT_PATH", Kind_Synchronization, true); ok {
		t.Fatal("able to find unset environment variable")
	}
}

func TestSourceLookupSourceSpecificExists(t *testing.T) {
	if value, ok := getEnvironmentVariable("DOCKER_CONTEXT", Kind_Forwarding, true); !ok {
		t.Fatal("unable to find source-specific value")
	} else if value != sourceSpecificDockerContext {
		t.Fatal("source-specific value does not match expected")
	}
}

func TestSourceLookupOnlyDefaultExists(t *testing.T) {
	if value, ok := getEnvironmentVariable("DOCKER_TLS_VERIFY", Kind_Forwarding, true); !ok {
		t.Fatal("unable to find non-endpoint-specific value for source")
	} else if value != defaultDockerTLSVerify {
		t.Fatal("non-endpoint-specific value does not match expected")
	}
}

func TestSourceLookupNeitherExists(t *testing.T) {
	if _, ok := getEnvironmentVariable("DOCKER_CERT_PATH", Kind_Forwarding, true); ok {
		t.Fatal("able to find unset environment variable")
	}
}

func TestDestinationLookupDestinationSpecificExists(t *testing.T) {
	if value, ok := getEnvironmentVariable("DOCKER_TLS_VERIFY", Kind_Forwarding, false); !ok {
		t.Fatal("unable to find destination-specific value")
	} else if value != destinationSpecificDockerTLSVerify {
		t.Fatal("destination-specific value does not match expected")
	}
}

func TestDestinationLookupOnlyDefaultExists(t *testing.T) {
	if value, ok := getEnvironmentVariable("DOCKER_HOST", Kind_Forwarding, false); !ok {
		t.Fatal("unable to find non-endpoint-specific value for source")
	} else if value != defaultDockerHost {
		t.Fatal("non-endpoint-specific value does not match expected")
	}
}

func TestDestinationLookupNeitherExists(t *testing.T) {
	if _, ok := getEnvironmentVariable("DOCKER_CERT_PATH", Kind_Forwarding, true); ok {
		t.Fatal("able to find unset environment variable")
	}
}
