package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/integration/fixtures/constants"

	"github.com/pkg/errors"

	"github.com/google/uuid"

	"github.com/mutagen-io/mutagen/pkg/integration/protocols/netpipe"
	"github.com/mutagen-io/mutagen/pkg/prompt"
	"github.com/mutagen-io/mutagen/pkg/selection"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/url"
)

func waitForSuccessfulSynchronizationCycle(sessionId string, allowConflicts, allowProblems bool) error {
	// Create a session selection specification.
	selection := &selection.Selection{
		Specifications: []string{sessionId},
	}

	// Perform waiting.
	var previousStateIndex uint64
	var states []*synchronization.State
	var err error
	for {
		previousStateIndex, states, err = synchronizationManager.List(selection, previousStateIndex)
		if err != nil {
			return errors.Wrap(err, "unable to list session states")
		} else if len(states) != 1 {
			return errors.New("invalid number of session states returned")
		} else if states[0].SuccessfulSynchronizationCycles > 0 {
			if !allowProblems && (len(states[0].AlphaProblems) > 0 || len(states[0].BetaProblems) > 0) {
				return errors.New("problems detected (and disallowed)")
			} else if !allowConflicts && len(states[0].Conflicts) > 0 {
				return errors.New("conflicts detected (and disallowed)")
			}
			return nil
		}
	}
}

func testSessionLifecycle(prompter string, alpha, beta *url.URL, configuration *synchronization.Configuration, allowConflicts, allowProblems bool) error {
	// Create a session.
	sessionId, err := synchronizationManager.Create(
		alpha, beta,
		configuration,
		&synchronization.Configuration{},
		&synchronization.Configuration{},
		"testSynchronizationSession",
		nil,
		false,
		prompter,
	)
	if err != nil {
		return errors.Wrap(err, "unable to create session")
	}

	// Wait for the session to have at least one successful synchronization
	// cycle.
	// TODO: Should we add a timeout on this?
	if err := waitForSuccessfulSynchronizationCycle(sessionId, allowConflicts, allowProblems); err != nil {
		return errors.Wrap(err, "unable to wait for successful synchronization")
	}

	// TODO: Add hook for verifying file contents.

	// TODO: Add hook for verifying presence/absence of particular
	// conflicts/problems and remove that monitoring from
	// waitForSuccessfulSynchronizationCycle (maybe have it pass back the
	// relevant state).

	// Create a session selection specification.
	selection := &selection.Selection{
		Specifications: []string{sessionId},
	}

	// Pause the session.
	if err := synchronizationManager.Pause(selection, ""); err != nil {
		return errors.Wrap(err, "unable to pause session")
	}

	// Resume the session.
	if err := synchronizationManager.Resume(selection, ""); err != nil {
		return errors.Wrap(err, "unable to resume session")
	}

	// Wait for the session to have at least one additional synchronization
	// cycle.
	// TODO: Should we add a timeout on this?
	if err := waitForSuccessfulSynchronizationCycle(sessionId, allowConflicts, allowProblems); err != nil {
		return errors.Wrap(err, "unable to wait for additional synchronization")
	}

	// Attempt an additional resume (this should be a no-op).
	if err := synchronizationManager.Resume(selection, ""); err != nil {
		return errors.Wrap(err, "unable to perform additional resume")
	}

	// Terminate the session.
	if err := synchronizationManager.Terminate(selection, ""); err != nil {
		return errors.Wrap(err, "unable to terminate session")
	}

	// TODO: Verify that cleanup took place.

	// Success.
	return nil
}

func TestSynchronizationBothRootsNil(t *testing.T) {
	// Allow this test to run in parallel.
	t.Parallel()

	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_end_to_end")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Calculate alpha and beta paths.
	alphaRoot := filepath.Join(directory, "alpha")
	betaRoot := filepath.Join(directory, "beta")

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{Path: betaRoot}

	// Compute configuration. We use defaults for everything.
	configuration := &synchronization.Configuration{}

	// Test the session lifecycle.
	if err := testSessionLifecycle("", alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

func TestSynchronizationGOROOTSrcToBeta(t *testing.T) {
	// Check the end-to-end test mode and compute the source synchronization
	// root accordingly. If no mode has been specified, then skip the test.
	endToEndTestMode := os.Getenv("MUTAGEN_TEST_END_TO_END")
	var sourceRoot string
	if endToEndTestMode == "" {
		t.Skip()
	} else if endToEndTestMode == "full" {
		sourceRoot = filepath.Join(runtime.GOROOT(), "src")
	} else if endToEndTestMode == "slim" {
		sourceRoot = filepath.Join(runtime.GOROOT(), "src", "bufio")
	} else {
		t.Fatal("unknown end-to-end test mode specified:", endToEndTestMode)
	}

	// Allow the test to run in parallel.
	t.Parallel()

	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_end_to_end")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Calculate alpha and beta paths.
	alphaRoot := sourceRoot
	betaRoot := filepath.Join(directory, "beta")

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{Path: betaRoot}

	// Compute configuration. We use defaults for everything.
	configuration := &synchronization.Configuration{}

	// Test the session lifecycle.
	if err := testSessionLifecycle("", alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

func TestSynchronizationGOROOTSrcToAlpha(t *testing.T) {
	// Check the end-to-end test mode and compute the source synchronization
	// root accordingly. If no mode has been specified, then skip the test.
	endToEndTestMode := os.Getenv("MUTAGEN_TEST_END_TO_END")
	var sourceRoot string
	if endToEndTestMode == "" {
		t.Skip()
	} else if endToEndTestMode == "full" {
		sourceRoot = filepath.Join(runtime.GOROOT(), "src")
	} else if endToEndTestMode == "slim" {
		sourceRoot = filepath.Join(runtime.GOROOT(), "src", "bufio")
	} else {
		t.Fatal("unknown end-to-end test mode specified:", endToEndTestMode)
	}

	// Allow the test to run in parallel.
	t.Parallel()

	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_end_to_end")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Calculate alpha and beta paths.
	alphaRoot := filepath.Join(directory, "alpha")
	betaRoot := sourceRoot

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{Path: betaRoot}

	// Compute configuration. We use defaults for everything.
	configuration := &synchronization.Configuration{}

	// Test the session lifecycle.
	if err := testSessionLifecycle("", alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

func TestSynchronizationGOROOTSrcToBetaInMemory(t *testing.T) {
	// Check the end-to-end test mode and compute the source synchronization
	// root accordingly. If no mode has been specified, then skip the test.
	endToEndTestMode := os.Getenv("MUTAGEN_TEST_END_TO_END")
	var sourceRoot string
	if endToEndTestMode == "" {
		t.Skip()
	} else if endToEndTestMode == "full" {
		sourceRoot = filepath.Join(runtime.GOROOT(), "src")
	} else if endToEndTestMode == "slim" {
		sourceRoot = filepath.Join(runtime.GOROOT(), "src", "bufio")
	} else {
		t.Fatal("unknown end-to-end test mode specified:", endToEndTestMode)
	}

	// Allow the test to run in parallel.
	t.Parallel()

	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_end_to_end")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Calculate alpha and beta paths.
	alphaRoot := sourceRoot
	betaRoot := filepath.Join(directory, "beta")

	// Compute alpha and beta URLs. We use a special protocol with a custom
	// handler to indicate an in-memory connection.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{
		Protocol: netpipe.Protocol_Netpipe,
		Path:     betaRoot,
	}

	// Compute configuration. We use defaults for everything.
	configuration := &synchronization.Configuration{}

	// Test the session lifecycle.
	if err := testSessionLifecycle("", alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

func TestSynchronizationGOROOTSrcToBetaOverSSH(t *testing.T) {
	// If localhost SSH support isn't available, then skip this test.
	if os.Getenv("MUTAGEN_TEST_SSH") != "true" {
		t.Skip()
	}

	// Check the end-to-end test mode and compute the source synchronization
	// root accordingly. If no mode has been specified, then skip the test.
	endToEndTestMode := os.Getenv("MUTAGEN_TEST_END_TO_END")
	var sourceRoot string
	if endToEndTestMode == "" {
		t.Skip()
	} else if endToEndTestMode == "full" {
		sourceRoot = filepath.Join(runtime.GOROOT(), "src")
	} else if endToEndTestMode == "slim" {
		sourceRoot = filepath.Join(runtime.GOROOT(), "src", "bufio")
	} else {
		t.Fatal("unknown end-to-end test mode specified:", endToEndTestMode)
	}

	// Allow the test to run in parallel.
	t.Parallel()

	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_end_to_end")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Calculate alpha and beta paths.
	alphaRoot := sourceRoot
	betaRoot := filepath.Join(directory, "beta")

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{
		Protocol: url.Protocol_SSH,
		Host:     "localhost",
		Path:     betaRoot,
	}

	// Compute configuration. We use defaults for everything.
	configuration := &synchronization.Configuration{}

	// Test the session lifecycle.
	if err := testSessionLifecycle("", alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

// testWindowsDockerTransportPrompter is a prompt.Prompter implementation that
// will answer "yes" to all prompts. It's needed to confirm container restart
// behavior in the Docker transport on Windows.
type testWindowsDockerTransportPrompter struct{}

func (t *testWindowsDockerTransportPrompter) Message(_ string) error {
	return nil
}

func (t *testWindowsDockerTransportPrompter) Prompt(_ string) (string, error) {
	return "yes", nil
}

func TestSynchronizationGOROOTSrcToBetaOverDocker(t *testing.T) {
	// If Docker test support isn't available, then skip this test.
	if os.Getenv("MUTAGEN_TEST_DOCKER") != "true" {
		t.Skip()
	}

	// Check the end-to-end test mode and compute the source synchronization
	// root accordingly. If no mode has been specified, then skip the test.
	endToEndTestMode := os.Getenv("MUTAGEN_TEST_END_TO_END")
	var sourceRoot string
	if endToEndTestMode == "" {
		t.Skip()
	} else if endToEndTestMode == "full" {
		sourceRoot = filepath.Join(runtime.GOROOT(), "src")
	} else if endToEndTestMode == "slim" {
		sourceRoot = filepath.Join(runtime.GOROOT(), "src", "bufio")
	} else {
		t.Fatal("unknown end-to-end test mode specified:", endToEndTestMode)
	}

	// Allow the test to run in parallel.
	t.Parallel()

	// If we're on Windows, register a prompter that will answer yes to
	// questions about stoping and restarting containers.
	var prompter string
	if runtime.GOOS == "windows" {
		if p, err := prompt.RegisterPrompter(&testWindowsDockerTransportPrompter{}); err != nil {
			t.Fatal("unable to register prompter:", err)
		} else {
			prompter = p
			defer prompt.UnregisterPrompter(prompter)
		}
	}

	// Create a unique directory name for synchronization into the container. We
	// don't clean it up, because it will be wiped out when the test container
	// is deleted.
	randomUUID, err := uuid.NewRandom()
	if err != nil {
		t.Fatal("unable to create random directory UUID:", err)
	}

	// Calculate alpha and beta paths.
	alphaRoot := sourceRoot
	betaRoot := "~/" + randomUUID.String()

	// Grab Docker environment variables.
	environment := make(map[string]string, len(url.DockerEnvironmentVariables))
	for _, variable := range url.DockerEnvironmentVariables {
		environment[variable] = os.Getenv(variable)
	}

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{
		Protocol:    url.Protocol_Docker,
		User:        os.Getenv("MUTAGEN_TEST_DOCKER_USERNAME"),
		Host:        os.Getenv("MUTAGEN_TEST_DOCKER_CONTAINER_NAME"),
		Path:        betaRoot,
		Environment: environment,
	}

	// Verify that the beta URL is valid (this will validate the test
	// environment variables as well).
	if err := betaURL.EnsureValid(); err != nil {
		t.Fatal("beta URL is invalid:", err)
	}

	// Compute configuration. We use defaults for everything.
	configuration := &synchronization.Configuration{}

	// Test the session lifecycle.
	if err := testSessionLifecycle(prompter, alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

func TestForwardingToHTTPDemo(t *testing.T) {
	// If Docker test support isn't available, then skip this test.
	if os.Getenv("MUTAGEN_TEST_DOCKER") != "true" {
		t.Skip()
	}

	// Allow the test to run in parallel.
	t.Parallel()

	// If we're on Windows, register a prompter that will answer yes to
	// questions about stoping and restarting containers.
	var prompter string
	if runtime.GOOS == "windows" {
		if p, err := prompt.RegisterPrompter(&testWindowsDockerTransportPrompter{}); err != nil {
			t.Fatal("unable to register prompter:", err)
		} else {
			prompter = p
			defer prompt.UnregisterPrompter(prompter)
		}
	}

	// Pick a local listener address.
	listenerProtocol := "tcp"
	listenerAddress := "localhost:7070"

	// Grab Docker environment variables.
	environment := make(map[string]string, len(url.DockerEnvironmentVariables))
	for _, variable := range url.DockerEnvironmentVariables {
		environment[variable] = os.Getenv(variable)
	}

	// Compute source and destination URLs.
	source := &url.URL{
		Kind:     url.Kind_Forwarding,
		Protocol: url.Protocol_Local,
		Path:     listenerProtocol + ":" + listenerAddress,
	}
	destination := &url.URL{
		Kind:     url.Kind_Forwarding,
		Protocol: url.Protocol_Docker,
		User:     os.Getenv("MUTAGEN_TEST_DOCKER_USERNAME"),
		Host:     os.Getenv("MUTAGEN_TEST_DOCKER_CONTAINER_NAME"),
		Path:     "tcp:" + constants.HTTPDemoBindAddress,
	}

	// Verify that the destination URL is valid (this will validate the test
	// environment variables as well).
	if err := destination.EnsureValid(); err != nil {
		t.Fatal("beta URL is invalid:", err)
	}

	// Create a function to perform a simple HTTP request and ensure that the
	// returned contents are as expected.
	performHTTPRequest := func() error {
		// Perform the request and defer closure of the response body.
		response, err := http.Get(fmt.Sprintf("http://%s/", listenerAddress))
		if err != nil {
			return errors.Wrap(err, "unable to perform HTTP GET")
		}
		defer response.Body.Close()

		// Read the full body.
		message, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return errors.Wrap(err, "unable to read response body")
		}

		// Compare the message.
		if string(message) != constants.HTTPDemoResponse {
			return errors.New("response does not match expected")
		}

		// Success.
		return nil
	}

	// Create a session.
	sessionID, err := forwardingManager.Create(
		source,
		destination,
		&forwarding.Configuration{},
		&forwarding.Configuration{},
		&forwarding.Configuration{},
		"testForwardingSession",
		nil,
		false,
		prompter,
	)
	if err != nil {
		t.Fatal("unable to create session:", err)
	}

	// Attempt server read.
	// TODO: Attempt a more complicated exchange here. Maybe gRPC?
	if err := performHTTPRequest(); err != nil {
		t.Error("error performing forwarded HTTP request:", err)
	}

	// Create a session selection specification.
	selection := &selection.Selection{
		Specifications: []string{sessionID},
	}

	// Pause the session.
	if err := forwardingManager.Pause(selection, ""); err != nil {
		t.Error("unable to pause session:", err)
	}

	// Resume the session.
	if err := forwardingManager.Resume(selection, ""); err != nil {
		t.Error("unable to resume session:", err)
	}

	// Attempt server read.
	// TODO: Attempt a more complicated exchange here. Maybe gRPC?
	if err := performHTTPRequest(); err != nil {
		t.Error("error performing forwarded HTTP request:", err)
	}

	// Attempt an additional resume (this should be a no-op).
	if err := forwardingManager.Resume(selection, ""); err != nil {
		t.Error("unable to perform additional resume:", err)
	}

	// Terminate the session.
	if err := forwardingManager.Terminate(selection, ""); err != nil {
		t.Error("unable to terminate session:", err)
	}

	// TODO: Verify that cleanup took place.
}

// TODO: Add forwarding tests using the netpipe protocol.
