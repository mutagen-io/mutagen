package integration

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"

	"github.com/google/uuid"

	"github.com/havoc-io/mutagen/pkg/agent"
	"github.com/havoc-io/mutagen/pkg/daemon"
	"github.com/havoc-io/mutagen/pkg/filesystem"
	"github.com/havoc-io/mutagen/pkg/local"
	"github.com/havoc-io/mutagen/pkg/prompt"
	"github.com/havoc-io/mutagen/pkg/session"
	"github.com/havoc-io/mutagen/pkg/url"

	// Explicitly import packages that need to register protocol handlers.
	_ "github.com/havoc-io/mutagen/pkg/docker"
	_ "github.com/havoc-io/mutagen/pkg/local"
	_ "github.com/havoc-io/mutagen/pkg/ssh"
)

// daemonLock is the daemon lock manager.
var daemonLock *daemon.Lock

// sessionManager is the session manager.
var sessionManager *session.Manager

func init() {
	// Copy the agent bundle for testing.
	// HACK: We're relying on the fact that Go will clean this up when it
	// removes the testing temporary directory.
	if err := agent.CopyBundleForTesting(); err != nil {
		panic(errors.Wrap(err, "unable to copy agent bundle for testing"))
	}

	// Acquire the daemon lock.
	if l, err := daemon.AcquireLock(); err != nil {
		panic(errors.Wrap(err, "unable to acquire daemon lock"))
	} else {
		daemonLock = l
	}

	// Create a session manager.
	if m, err := session.NewManager(); err != nil {
		panic(errors.Wrap(err, "unable to create session manager"))
	} else {
		sessionManager = m
	}

	// Perform housekeeping.
	agent.Housekeep()
	local.HousekeepCaches()
	local.HousekeepStaging()
}

func waitForSuccessfulSynchronizationCycle(sessionId string, allowConflicts, allowProblems bool) error {
	// Create a session specification.
	specification := []string{sessionId}

	// Perform waiting.
	var previousStateIndex uint64
	var states []*session.State
	var err error
	for {
		previousStateIndex, states, err = sessionManager.List(previousStateIndex, specification)
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

func testSessionLifecycle(prompter string, alpha, beta *url.URL, configuration *session.Configuration, allowConflicts, allowProblems bool) error {
	// Create a session.
	sessionId, err := sessionManager.Create(alpha, beta, configuration, prompter)
	if err != nil {
		return errors.Wrap(err, "unable to create session")
	}

	// Create a session specification.
	specification := []string{sessionId}

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

	// Pause the session.
	if err := sessionManager.Pause(specification, ""); err != nil {
		return errors.Wrap(err, "unable to pause session")
	}

	// Resume the session.
	if err := sessionManager.Resume(specification, ""); err != nil {
		return errors.Wrap(err, "unable to resume session")
	}

	// Wait for the session to have at least one additional synchronization
	// cycle.
	// TODO: Should we add a timeout on this?
	if err := waitForSuccessfulSynchronizationCycle(sessionId, allowConflicts, allowProblems); err != nil {
		return errors.Wrap(err, "unable to wait for additional synchronization")
	}

	// Attempt an additional resume (this should be a no-op).
	if err := sessionManager.Resume(specification, ""); err != nil {
		return errors.Wrap(err, "unable to perform additional resume")
	}

	// Terminate the session.
	if err := sessionManager.Terminate(specification, ""); err != nil {
		return errors.Wrap(err, "unable to terminate session")
	}

	// TODO: Verify that cleanup took place.

	// Success.
	return nil
}

func TestSessionBothRootsNil(t *testing.T) {
	// If end-to-end tests haven't been enabled, then skip this test.
	if os.Getenv("MUTAGEN_TEST_END_TO_END") != "true" {
		t.Skip()
	}

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

	// Compute configuration.
	// HACK: The notify package has a race condition on Windows that the race
	// detector catches, so force polling there for now during tests. Force
	// polling on macOS as well since notify seems flaky in tests there as well.
	configuration := &session.Configuration{}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		configuration.WatchMode = filesystem.WatchMode_WatchForcePoll
	}

	// Test the session lifecycle.
	if err := testSessionLifecycle("", alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

func TestSessionGOROOTSrcToBeta(t *testing.T) {
	// If end-to-end tests haven't been enabled, then skip this test.
	if os.Getenv("MUTAGEN_TEST_END_TO_END") != "true" {
		t.Skip()
	}

	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_end_to_end")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Calculate alpha and beta paths.
	alphaRoot := filepath.Join(runtime.GOROOT(), "src")
	betaRoot := filepath.Join(directory, "beta")

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{Path: betaRoot}

	// Compute configuration.
	// HACK: The notify package has a race condition on Windows that the race
	// detector catches, so force polling there for now during tests. Force
	// polling on macOS as well since notify seems flaky in tests there as well.
	configuration := &session.Configuration{}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		configuration.WatchMode = filesystem.WatchMode_WatchForcePoll
	}

	// Test the session lifecycle.
	if err := testSessionLifecycle("", alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

func TestSessionGOROOTSrcToAlpha(t *testing.T) {
	// If end-to-end tests haven't been enabled, then skip this test.
	if os.Getenv("MUTAGEN_TEST_END_TO_END") != "true" {
		t.Skip()
	}

	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_end_to_end")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Calculate alpha and beta paths.
	alphaRoot := filepath.Join(directory, "alpha")
	betaRoot := filepath.Join(runtime.GOROOT(), "src")

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{Path: betaRoot}

	// Compute configuration.
	// HACK: The notify package has a race condition on Windows that the race
	// detector catches, so force polling there for now during tests. Force
	// polling on macOS as well since notify seems flaky in tests there as well.
	configuration := &session.Configuration{}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		configuration.WatchMode = filesystem.WatchMode_WatchForcePoll
	}

	// Test the session lifecycle.
	if err := testSessionLifecycle("", alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

func TestSessionGOROOTSrcToBetaInMemory(t *testing.T) {
	// If end-to-end tests haven't been enabled, then skip this test.
	if os.Getenv("MUTAGEN_TEST_END_TO_END") != "true" {
		t.Skip()
	}

	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_end_to_end")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Calculate alpha and beta paths.
	alphaRoot := filepath.Join(runtime.GOROOT(), "src")
	betaRoot := filepath.Join(directory, "beta")

	// Compute alpha and beta URLs. We use a special protocol with a custom
	// handler to indicate an in-memory connection.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{
		Protocol: inMemoryProtocol,
		Path:     betaRoot,
	}

	// Compute configuration.
	// HACK: The notify package has a race condition on Windows that the race
	// detector catches, so force polling there for now during tests. Force
	// polling on macOS as well since notify seems flaky in tests there as well.
	configuration := &session.Configuration{}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		configuration.WatchMode = filesystem.WatchMode_WatchForcePoll
	}

	// Test the session lifecycle.
	if err := testSessionLifecycle("", alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

func TestSessionGOROOTSrcToBetaOverSSH(t *testing.T) {
	// If end-to-end tests haven't been enabled, then skip this test.
	if os.Getenv("MUTAGEN_TEST_END_TO_END") != "true" {
		t.Skip()
	}

	// If localhost SSH support isn't available, then skip this test.
	if os.Getenv("MUTAGEN_TEST_SSH") != "true" {
		t.Skip()
	}

	// Create a temporary directory and defer its cleanup.
	directory, err := ioutil.TempDir("", "mutagen_end_to_end")
	if err != nil {
		t.Fatal("unable to create temporary directory:", err)
	}
	defer os.RemoveAll(directory)

	// Calculate alpha and beta paths.
	alphaRoot := filepath.Join(runtime.GOROOT(), "src")
	betaRoot := filepath.Join(directory, "beta")

	// Compute alpha and beta URLs.
	alphaURL := &url.URL{Path: alphaRoot}
	betaURL := &url.URL{
		Protocol: url.Protocol_SSH,
		Hostname: "localhost",
		Path:     betaRoot,
	}

	// Compute configuration.
	// HACK: The notify package has a race condition on Windows that the race
	// detector catches, so force polling there for now during tests. Force
	// polling on macOS as well since notify seems flaky in tests there as well.
	configuration := &session.Configuration{}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		configuration.WatchMode = filesystem.WatchMode_WatchForcePoll
	}

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

func TestSessionGOROOTSrcToBetaOverDocker(t *testing.T) {
	// If end-to-end tests haven't been enabled, then skip this test.
	if os.Getenv("MUTAGEN_TEST_END_TO_END") != "true" {
		t.Skip()
	}

	// If Docker test support isn't available, then skip this test.
	if os.Getenv("MUTAGEN_TEST_DOCKER") != "true" {
		t.Skip()
	}

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
	alphaRoot := filepath.Join(runtime.GOROOT(), "src")
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
		Username:    os.Getenv("MUTAGEN_TEST_DOCKER_USERNAME"),
		Hostname:    os.Getenv("MUTAGEN_TEST_DOCKER_CONTAINER_NAME"),
		Path:        betaRoot,
		Environment: environment,
	}

	// Verify that the beta URL is valid (this will validate the test
	// environment variables as well).
	if err := betaURL.EnsureValid(); err != nil {
		t.Fatal("beta URL is invalid:", err)
	}

	// Compute configuration.
	// HACK: The notify package has a race condition on Windows that the race
	// detector catches, so force polling there for now during tests. Force
	// polling on macOS as well since notify seems flaky in tests there as well.
	configuration := &session.Configuration{}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		configuration.WatchMode = filesystem.WatchMode_WatchForcePoll
	}

	// Test the session lifecycle.
	if err := testSessionLifecycle(prompter, alphaURL, betaURL, configuration, false, false); err != nil {
		t.Fatal("session lifecycle test failed:", err)
	}
}

// TODO: Implement end-to-end tests that work via the gRPC service endpoints.
// This will obviously require setting up the whole service architecture. Maybe
// we can modify the session service to take a session manager as an argument
// (instead of creating it internally) so that we don't need to tear down the
// other manager before trying the gRPC version? Manager is already safe for
// concurrent usage.
