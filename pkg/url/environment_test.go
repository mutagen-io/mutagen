package url

import (
	"testing"
)

const (
	// alphaSpecificDockerHostEnvironmentVariable is the name of the
	// alpha-specific DOCKER_HOST environment variable.
	alphaSpecificDockerHostEnvironmentVariable = "MUTAGEN_ALPHA_DOCKER_HOST"
	// betaSpecificDockerTLSVerifyEnvironmentVariable is the name of the
	// beta-specific DOCKER_TLS_VERIFY environment variable.
	betaSpecificDockerTLSVerifyEnvironmentVariable = "MUTAGEN_BETA_DOCKER_TLS_VERIFY"
	// defaultDockerHost is the non-endpoint-specific value for the DOCKER_HOST
	// environment variable.
	defaultDockerHost = "unix:///default/docker.sock"
	// alphaSpecificDockerHost is the alpha-specific value for the DOCKER_HOST
	// environment variable.
	alphaSpecificDockerHost = "unix:///alpha/docker.sock"
	// defaultDockerTLSVerify is the non-endpoint-specific value for the
	// DOCKER_TLS_VERIFY environment variable.
	defaultDockerTLSVerify = "sure!"
	// betaSpecificDockerTLSVerify is the beta-specific value for the
	// DOCKER_TLS_VERIFY environment variable.
	betaSpecificDockerTLSVerify = "true"
)

// mockEnvironment is a mock environment setup for use in testing.
var mockEnvironment = map[string]string{
	DockerHostEnvironmentVariable:                  defaultDockerHost,
	alphaSpecificDockerHostEnvironmentVariable:     alphaSpecificDockerHost,
	DockerTLSVerifyEnvironmentVariable:             defaultDockerTLSVerify,
	betaSpecificDockerTLSVerifyEnvironmentVariable: betaSpecificDockerTLSVerify,
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
	if value, ok := getEnvironmentVariable(DockerHostEnvironmentVariable, true); !ok {
		t.Fatal("unable to find alpha-specific value")
	} else if value != alphaSpecificDockerHost {
		t.Fatal("alpha-specific value does not match expected")
	}
}

func TestAlphaLookupOnlyDefaultExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerTLSVerifyEnvironmentVariable, true); !ok {
		t.Fatal("unable to find non-endpoint-specific value for alpha")
	} else if value != defaultDockerTLSVerify {
		t.Fatal("non-endpoint-specific value does not match expected")
	}
}

func TestAlphaLookupNeitherExists(t *testing.T) {
	if _, ok := getEnvironmentVariable(DockerCertPathEnvironmentVariable, true); ok {
		t.Fatal("able to find unset environment variable")
	}
}

func TestBetaLookupBetaSpecificExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerTLSVerifyEnvironmentVariable, false); !ok {
		t.Fatal("unable to find beta-specific value")
	} else if value != betaSpecificDockerTLSVerify {
		t.Fatal("beta-specific value does not match expected")
	}
}

func TestBetaLookupOnlyDefaultExists(t *testing.T) {
	if value, ok := getEnvironmentVariable(DockerHostEnvironmentVariable, false); !ok {
		t.Fatal("unable to find non-endpoint-specific value for alpha")
	} else if value != defaultDockerHost {
		t.Fatal("non-endpoint-specific value does not match expected")
	}
}

func TestBetaLookupNeitherExists(t *testing.T) {
	if _, ok := getEnvironmentVariable(DockerCertPathEnvironmentVariable, true); ok {
		t.Fatal("able to find unset environment variable")
	}
}
