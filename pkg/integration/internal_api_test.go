package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"

	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/local"
	"github.com/mutagen-io/mutagen/pkg/integration/fixtures/constants"
	"github.com/mutagen-io/mutagen/pkg/integration/protocols/netpipe"
	"github.com/mutagen-io/mutagen/pkg/prompting"
	"github.com/mutagen-io/mutagen/pkg/selection"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/url"
)

func waitForSuccessfulSynchronizationCycle(ctx context.Context, sessionID string, allowScanProblems, allowConflicts, allowTransitionProblems bool) error {
	// Create a session selection specification.
	selection := &selection.Selection{
		Specifications: []string{sessionID},
	}

	// Perform waiting.
	var previousStateIndex uint64
	var states []*synchronization.State
	var err error
	for {
		previousStateIndex, states, err = synchronizationManager.List(ctx, selection, previousStateIndex)
		if err != nil {
			return fmt.Errorf("unable to list session states: %w", err)
		} else if len(states) != 1 {
			return errors.New("invalid number of session states returned")
		} else if states[0].SuccessfulCycles > 0 {
			if !allowScanProblems && (len(states[0].AlphaScanProblems) > 0 || len(states[0].BetaScanProblems) > 0) {
				return errors.New("scan problems detected (and disallowed)")
			} else if !allowConflicts && len(states[0].Conflicts) > 0 {
				return errors.New("conflicts detected (and disallowed)")
			} else if !allowTransitionProblems && (len(states[0].AlphaTransitionProblems) > 0 || len(states[0].BetaTransitionProblems) > 0) {
				return errors.New("transition problems detected (and disallowed)")
			}
			return nil
		}
	}
}

func testSessionLifecycle(ctx context.Context, prompter string, alpha, beta *url.URL, configuration *synchronization.Configuration, allowScanProblems, allowConflicts, allowTransitionProblems bool) error {
	// Create a session.
	sessionID, err := synchronizationManager.Create(
		ctx,
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
		return fmt.Errorf("unable to create session: %w", err)
	}

	// Wait for the session to have at least one successful synchronization
	// cycle.
	// TODO: Should we add a timeout on this?
	if err := waitForSuccessfulSynchronizationCycle(ctx, sessionID, allowScanProblems, allowConflicts, allowTransitionProblems); err != nil {
		return fmt.Errorf("unable to wait for successful synchronization: %w", err)
	}

	// TODO: Add hook for verifying file contents.

	// TODO: Add hook for verifying presence/absence of particular
	// conflicts/problems and remove that monitoring from
	// waitForSuccessfulSynchronizationCycle (maybe have it pass back the
	// relevant state).

	// Create a session selection specification.
	selection := &selection.Selection{
		Specifications: []string{sessionID},
	}

	// Pause the session.
	if err := synchronizationManager.Pause(ctx, selection, ""); err != nil {
		return fmt.Errorf("unable to pause session: %w", err)
	}

	// Resume the session.
	if err := synchronizationManager.Resume(ctx, selection, ""); err != nil {
		return fmt.Errorf("unable to resume session: %w", err)
	}

	// Wait for the session to have at least one additional synchronization
	// cycle.
	if err := waitForSuccessfulSynchronizationCycle(ctx, sessionID, allowScanProblems, allowConflicts, allowTransitionProblems); err != nil {
		return fmt.Errorf("unable to wait for additional synchronization: %w", err)
	}

	// Attempt an additional resume (this should be a no-op).
	if err := synchronizationManager.Resume(ctx, selection, ""); err != nil {
		return fmt.Errorf("unable to perform additional resume: %w", err)
	}

	// Terminate the session.
	if err := synchronizationManager.Terminate(ctx, selection, ""); err != nil {
		return fmt.Errorf("unable to terminate session: %w", err)
	}

	// TODO: Verify that cleanup took place.

	// Success.
	return nil
}

func TestSynchronizationBothRootsNil(t *testing.T) {
	// Allow this test to run in parallel.
	t.Parallel()

	// Calculate alpha and beta paths.
	directory := t.TempDir()
	alphaRoot := filepath.Join(directory, "alpha")
	betaRoot := filepath.Join(directory, "beta")

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{Path: betaRoot}

	// Compute configuration. We use defaults for everything.
	configuration := &synchronization.Configuration{}

	// Test the session lifecycle.
	if err := testSessionLifecycle(context.Background(), "", alphaURL, betaURL, configuration, false, false, false); err != nil {
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

	// Calculate alpha and beta paths.
	alphaRoot := sourceRoot
	betaRoot := filepath.Join(t.TempDir(), "beta")

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{Path: betaRoot}

	// Compute configuration. We use defaults for everything.
	configuration := &synchronization.Configuration{}

	// Test the session lifecycle.
	if err := testSessionLifecycle(context.Background(), "", alphaURL, betaURL, configuration, false, false, false); err != nil {
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

	// Calculate alpha and beta paths.
	alphaRoot := filepath.Join(t.TempDir(), "alpha")
	betaRoot := sourceRoot

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{Path: betaRoot}

	// Compute configuration. We use defaults for everything.
	configuration := &synchronization.Configuration{}

	// Test the session lifecycle.
	if err := testSessionLifecycle(context.Background(), "", alphaURL, betaURL, configuration, false, false, false); err != nil {
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

	// Calculate alpha and beta paths.
	alphaRoot := sourceRoot
	betaRoot := filepath.Join(t.TempDir(), "beta")

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
	if err := testSessionLifecycle(context.Background(), "", alphaURL, betaURL, configuration, false, false, false); err != nil {
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

	// Calculate alpha and beta paths.
	alphaRoot := sourceRoot
	betaRoot := filepath.Join(t.TempDir(), "beta")

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
	if err := testSessionLifecycle(context.Background(), "", alphaURL, betaURL, configuration, false, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

// testWindowsDockerTransportPrompter is a prompting.Prompter implementation
// that will answer "yes" to all prompts. It's needed to confirm container
// restart behavior in the Docker transport on Windows.
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

	// If we're on a POSIX system, then allow this test to run concurrently with
	// other tests. On Windows, agent installation into Docker containers
	// requires temporarily halting the container, meaning that multiple
	// simultaneous Docker tests could conflict with each other, so we don't
	// allow Docker-based tests to run concurrently on Windows.
	if runtime.GOOS != "windows" {
		t.Parallel()
	}

	// If we're on Windows, register a prompter that will answer yes to
	// questions about stoping and restarting containers.
	var prompter string
	if runtime.GOOS == "windows" {
		if p, err := prompting.RegisterPrompter(&testWindowsDockerTransportPrompter{}); err != nil {
			t.Fatal("unable to register prompter:", err)
		} else {
			prompter = p
			defer prompting.UnregisterPrompter(prompter)
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
	if err := testSessionLifecycle(context.Background(), prompter, alphaURL, betaURL, configuration, false, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

func init() {
	// HACK: Disable lazy listener initialization since it makes test
	// coordination difficult.
	local.DisableLazyListenerInitialization = true
}

func TestForwardingToHTTPDemo(t *testing.T) {
	// If Docker test support isn't available, then skip this test.
	if os.Getenv("MUTAGEN_TEST_DOCKER") != "true" {
		t.Skip()
	}

	// If we're on a POSIX system, then allow this test to run concurrently with
	// other tests. On Windows, agent installation into Docker containers
	// requires temporarily halting the container, meaning that multiple
	// simultaneous Docker tests could conflict with each other, so we don't
	// allow Docker-based tests to run concurrently on Windows.
	if runtime.GOOS != "windows" {
		t.Parallel()
	}

	// If we're on Windows, register a prompter that will answer yes to
	// questions about stoping and restarting containers.
	var prompter string
	if runtime.GOOS == "windows" {
		if p, err := prompting.RegisterPrompter(&testWindowsDockerTransportPrompter{}); err != nil {
			t.Fatal("unable to register prompter:", err)
		} else {
			prompter = p
			defer prompting.UnregisterPrompter(prompter)
		}
	}

	// Pick a local listener address.
	listenerProtocol := "tcp"
	listenerAddress := "localhost:7070"

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
			return fmt.Errorf("unable to perform HTTP GET: %w", err)
		}
		defer response.Body.Close()

		// Read the full body.
		message, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("unable to read response body: %w", err)
		}

		// Compare the message.
		if string(message) != constants.HTTPDemoResponse {
			return errors.New("response does not match expected")
		}

		// Success.
		return nil
	}

	// Create a context to regulate the test.
	ctx := context.Background()

	// Create a forwarding session. Note that we've disabled lazy listener
	// initialization using a private API in the init function above, so we can
	// be sure that the listener has been established (with some non-empty
	// backlog) by the time creation is complete.
	sessionID, err := forwardingManager.Create(
		ctx,
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

	// Attempt an HTTP request.
	// TODO: Attempt a more complicated exchange here. Maybe gRPC?
	if err := performHTTPRequest(); err != nil {
		t.Error("error performing forwarded HTTP request:", err)
	}

	// Create a session selection specification.
	selection := &selection.Selection{
		Specifications: []string{sessionID},
	}

	// Pause the session.
	if err := forwardingManager.Pause(ctx, selection, ""); err != nil {
		t.Error("unable to pause session:", err)
	}

	// Resume the session.
	if err := forwardingManager.Resume(ctx, selection, ""); err != nil {
		t.Error("unable to resume session:", err)
	}

	// Attempt an HTTP request.
	// TODO: Attempt a more complicated exchange here. Maybe gRPC?
	if err := performHTTPRequest(); err != nil {
		t.Error("error performing forwarded HTTP request:", err)
	}

	// Attempt an additional resume (this should be a no-op).
	if err := forwardingManager.Resume(ctx, selection, ""); err != nil {
		t.Error("unable to perform additional resume:", err)
	}

	// Terminate the session.
	if err := forwardingManager.Terminate(ctx, selection, ""); err != nil {
		t.Error("unable to terminate session:", err)
	}

	// TODO: Verify that cleanup took place.
}

// TODO: Add forwarding tests using the netpipe protocol.
