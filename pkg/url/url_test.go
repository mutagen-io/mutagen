package url

import (
	"testing"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

// TestKindSupported tests that URL kind support detection works as expected.
func TestKindSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		kind     Kind
		expected bool
	}{
		{Kind_Synchronization, true},
		{Kind_Forwarding, true},
		{(Kind_Forwarding + 1), false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if supported := testCase.kind.Supported(); supported != testCase.expected {
			t.Errorf(
				"kind support (%t) does not match expected (%t)",
				supported,
				testCase.expected,
			)
		}
	}
}

func TestURLEnsureValidNilInvalid(t *testing.T) {
	var invalid *URL
	if invalid.EnsureValid() == nil {
		t.Error("nil URL marked as valid")
	}
}

func TestURLEnsureValidLocalUsernameInvalid(t *testing.T) {
	invalid := &URL{
		User: "george",
		Path: "some/path",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLForwardingEnsureValidForwardingEndpointRelativeLocalSocketInvalid(t *testing.T) {
	invalid := &URL{
		Kind: Kind_Forwarding,
		Path: "unix:relative/socket.sock",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLForwardingEnsureValidForwardingEndpointInvalidProtocolInvalid(t *testing.T) {
	invalid := &URL{
		Kind: Kind_Forwarding,
		Path: "tcp5:localhost:4420",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidLocalHostnameInvalid(t *testing.T) {
	invalid := &URL{
		Host: "somehost",
		Path: "some/path",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidLocalPortInvalid(t *testing.T) {
	invalid := &URL{
		Port: 22,
		Path: "some/path",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidLocalEmptyPathInvalid(t *testing.T) {
	invalid := &URL{}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidLocalRelativePathInvalid(t *testing.T) {
	invalid := &URL{
		Path: "relative/path",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidLocalEnvironmentVariablesInvalid(t *testing.T) {
	invalid := &URL{
		Path: "some/path",
		Environment: map[string]string{
			"key": "value",
		},
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidLocal(t *testing.T) {
	// Compute a normalized path.
	normalized, err := filesystem.Normalize("/some/path")
	if err != nil {
		t.Fatal("unable to normalize path:", err)
	}

	// Create and validate the URL.
	valid := &URL{Path: normalized}
	if err := valid.EnsureValid(); err != nil {
		t.Error("valid URL classified as invalid")
	}
}

func TestURLEnsureValidForwardingLocalTCP(t *testing.T) {
	valid := &URL{
		Kind: Kind_Forwarding,
		Path: "tcp:localhost:50505",
	}
	if err := valid.EnsureValid(); err != nil {
		t.Error("valid URL classified as invalid")
	}
}

func TestURLEnsureValidForwardingLocalUnixDomainSocket(t *testing.T) {
	// Compute a normalized Unix domain socket path.
	normalized, err := filesystem.Normalize("/socket/path.sock")
	if err != nil {
		t.Fatal("unable to normalize socket path:", err)
	}

	// Create and validate the URL.
	valid := &URL{
		Kind: Kind_Forwarding,
		Path: "unix:" + normalized,
	}
	if err := valid.EnsureValid(); err != nil {
		t.Error("valid URL classified as invalid")
	}
}

func TestURLEnsureValidSSHEmptyHostnameInvalid(t *testing.T) {
	invalid := &URL{
		Protocol: Protocol_SSH,
		Path:     "some/path",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidSSHLargePortInvalid(t *testing.T) {
	invalid := &URL{
		Protocol: Protocol_SSH,
		Host:     "washington",
		Port:     65536,
		Path:     "some/path",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidSSHEmptyPathInvalid(t *testing.T) {
	invalid := &URL{
		Protocol: Protocol_SSH,
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidSSHEnvironmentVariablesInvalid(t *testing.T) {
	invalid := &URL{
		Protocol: Protocol_SSH,
		User:     "george",
		Host:     "washington",
		Port:     22,
		Path:     "~/path",
		Environment: map[string]string{
			"key": "value",
		},
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidSSH(t *testing.T) {
	valid := &URL{
		Protocol: Protocol_SSH,
		User:     "george",
		Host:     "washington",
		Port:     22,
		Path:     "~/path",
	}
	if err := valid.EnsureValid(); err != nil {
		t.Error("valid URL classified as invalid")
	}
}

func TestURLEnsureValidDockerPortInvalid(t *testing.T) {
	invalid := &URL{
		Protocol: Protocol_Docker,
		User:     "george",
		Host:     "washington",
		Port:     50,
		Path:     "~/path",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidDockerEmptyPathInvalid(t *testing.T) {
	invalid := &URL{
		Protocol: Protocol_Docker,
		User:     "george",
		Host:     "washington",
		Path:     "",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidDockerBadPathInvalid(t *testing.T) {
	invalid := &URL{
		Protocol: Protocol_Docker,
		User:     "george",
		Host:     "washington",
		Path:     "$path",
	}
	if invalid.EnsureValid() == nil {
		t.Error("invalid URL classified as valid")
	}
}

func TestURLEnsureValidDockerHomeRelativePath(t *testing.T) {
	valid := &URL{
		Protocol: Protocol_Docker,
		User:     "george",
		Host:     "washington",
		Path:     "~/path",
	}
	if err := valid.EnsureValid(); err != nil {
		t.Error("valid URL classified as invalid")
	}
}

func TestURLEnsureValidDockerUserRelativePath(t *testing.T) {
	valid := &URL{
		Protocol: Protocol_Docker,
		User:     "george",
		Host:     "washington",
		Path:     "~otheruser/path",
	}
	if err := valid.EnsureValid(); err != nil {
		t.Error("valid URL classified as invalid")
	}
}

func TestURLEnsureValidDockerWindowsPath(t *testing.T) {
	valid := &URL{
		Protocol: Protocol_Docker,
		User:     "george",
		Host:     "washington",
		Path:     `C:\path`,
	}
	if err := valid.EnsureValid(); err != nil {
		t.Error("valid URL classified as invalid")
	}
}

func TestURLEnsureValidDocker(t *testing.T) {
	valid := &URL{
		Protocol: Protocol_Docker,
		User:     "george",
		Host:     "washington",
		Path:     "/path",
	}
	if err := valid.EnsureValid(); err != nil {
		t.Error("valid URL classified as invalid")
	}
}
