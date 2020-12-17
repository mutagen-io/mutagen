package url

import (
	"runtime"
	"testing"

	"github.com/mutagen-io/mutagen/pkg/filesystem"
)

type parseTestCase struct {
	raw      string
	kind     Kind
	first    bool
	fail     bool
	expected *URL
}

func (c *parseTestCase) run(t *testing.T) {
	// Mark this as a helper function to remove it from error traces.
	t.Helper()

	// Attempt to parse.
	url, err := Parse(c.raw, c.kind, c.first)
	if err != nil {
		if !c.fail {
			t.Fatal("parsing failed when it should have succeeded:", err)
		}
		return
	} else if c.fail {
		t.Fatal("parsing should have failed but did not")
	}

	// Verify kind.
	if url.Kind != c.expected.Kind {
		t.Error("kind mismatch:", url.Kind, "!=", c.expected.Kind)
	}

	// Verify protocol.
	if url.Protocol != c.expected.Protocol {
		t.Error("protocol mismatch:", url.Protocol, "!=", c.expected.Protocol)
	}

	// Verify username.
	if url.User != c.expected.User {
		t.Error("username mismatch:", url.User, "!=", c.expected.User)
	}

	// Verify hostname.
	if url.Host != c.expected.Host {
		t.Error("hostname mismatch:", url.Host, "!=", c.expected.Host)
	}

	// Verify port.
	if url.Port != c.expected.Port {
		t.Error("port mismatch:", url.Port, "!=", c.expected.Port)
	}

	// Verify path.
	if url.Path != c.expected.Path {
		t.Error("path mismatch:", url.Path, "!=", c.expected.Path)
	}

	// Verify environment variables.
	if len(url.Environment) != len(c.expected.Environment) {
		t.Error("environment length mismatch:", len(url.Environment), "!=", len(c.expected.Environment))
	} else {
		for ek, ev := range c.expected.Environment {
			if v, ok := url.Environment[ek]; !ok {
				t.Error("expected environment variable", ek, "not in URL environment")
			} else if v != ev {
				t.Error("environment variable", ek, "value does not match expected:", v, "!=", ev)
			}
		}
	}
}

func TestParseEmptyInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  "",
		fail: true,
	}
	test.run(t)
}

func TestParseLocalPathRelative(t *testing.T) {
	// Compute the normalized form of a relative path.
	path := "relative/path"
	normalized, err := filesystem.Normalize(path)
	if err != nil {
		t.Fatal("unable to normalize relative path:", err)
	}

	// Create and run the test case.
	test := parseTestCase{
		raw: path,
		expected: &URL{
			Protocol: Protocol_Local,
			User:     "",
			Host:     "",
			Port:     0,
			Path:     normalized,
		},
	}
	test.run(t)
}

func TestParseLocalPathAbsolute(t *testing.T) {
	// Compute the normalized form of an absolute path.
	path := "/this/is/a:path"
	normalized, err := filesystem.Normalize(path)
	if err != nil {
		t.Fatal("unable to normalize absolute path:", err)
	}

	// Create and run the test case.
	test := parseTestCase{
		raw: path,
		expected: &URL{
			Protocol: Protocol_Local,
			User:     "",
			Host:     "",
			Port:     0,
			Path:     normalized,
		},
	}
	test.run(t)
}

func TestParseForwardingLocalTCP(t *testing.T) {
	test := parseTestCase{
		raw:  "tcp:localhost:5050",
		kind: Kind_Forwarding,
		expected: &URL{
			Kind:     Kind_Forwarding,
			Protocol: Protocol_Local,
			User:     "",
			Host:     "",
			Port:     0,
			Path:     "tcp:localhost:5050",
		},
	}
	test.run(t)
}

func TestParseForwardingLocalUnixRelativeSocket(t *testing.T) {
	// Compute the normalized form of a relative socket path.
	path := "relative/path/to/socket.sock"
	normalized, err := filesystem.Normalize(path)
	if err != nil {
		t.Fatal("unable to normalize relative socket path:", err)
	}

	// Create and run the test case.
	test := parseTestCase{
		raw:  "unix:" + path,
		kind: Kind_Forwarding,
		expected: &URL{
			Kind:     Kind_Forwarding,
			Protocol: Protocol_Local,
			User:     "",
			Host:     "",
			Port:     0,
			Path:     "unix:" + normalized,
		},
	}
	test.run(t)
}

func TestParseForwardingLocalUnixAbsoluteSocket(t *testing.T) {
	// Compute the normalized form of a absolute socket path.
	path := "/path/to/socket.sock"
	normalized, err := filesystem.Normalize(path)
	if err != nil {
		t.Fatal("unable to normalize absolute socket path:", err)
	}

	// Create and run the test case.
	test := parseTestCase{
		raw:  "unix:" + path,
		kind: Kind_Forwarding,
		expected: &URL{
			Kind:     Kind_Forwarding,
			Protocol: Protocol_Local,
			User:     "",
			Host:     "",
			Port:     0,
			Path:     "unix:" + normalized,
		},
	}
	test.run(t)
}

func TestParseLocalPathWithAtSymbol(t *testing.T) {
	// Compute the normalized form of the path.
	path := "/some@path"
	normalized, err := filesystem.Normalize(path)
	if err != nil {
		t.Fatal("unable to normalize path:", err)
	}

	// Create and run the test case.
	test := parseTestCase{
		raw: path,
		expected: &URL{
			Protocol: Protocol_Local,
			User:     "",
			Host:     "",
			Port:     0,
			Path:     normalized,
		},
	}
	test.run(t)
}

func TestParsePOSIXSCPSSHWindowsLocal(t *testing.T) {
	// Compute the expected URL.
	raw := "C:/local/path"
	var expected *URL
	if runtime.GOOS == "windows" {
		if normalized, err := filesystem.Normalize(raw); err != nil {
			t.Fatal("unable to normalize path:", err)
		} else {
			expected = &URL{
				Path: normalized,
			}
		}
	} else {
		expected = &URL{
			Protocol: Protocol_SSH,
			Host:     "C",
			Path:     "/local/path",
		}
	}

	// Create and run the test case.
	test := &parseTestCase{
		raw:      raw,
		expected: expected,
	}
	test.run(t)
}

func TestParseSCPSSHEmptyHostnameInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  ":path",
		fail: true,
	}
	test.run(t)
}

func TestParseSCPSSHEmptyHostnameAndPathInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  ":",
		fail: true,
	}
	test.run(t)
}

func TestParseSCPSSHUsernameEmptyHostnameInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  "user@:path",
		fail: true,
	}
	test.run(t)
}

func TestParseSCPSSHUsernameEmptyPathInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:",
		fail: true,
	}
	test.run(t)
}

func TestParseSCPSSHEmptyUsernameInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  "@host:path",
		fail: true,
	}
	test.run(t)
}

func TestParseSCPSSHUsernamePortEmptyPathInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:5332:",
		fail: true,
	}
	test.run(t)
}

func TestParseSCPSSHHostnameEmptyPathInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  "host:",
		fail: true,
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnamePathEmptyPortInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host::path",
		fail: true,
	}
	test.run(t)
}

func TestParseSCPSSHHostnamePath(t *testing.T) {
	test := parseTestCase{
		raw: "host:path",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "",
			Host:     "host",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseForwardingSCPSSHHostnameTCPEndpoint(t *testing.T) {
	test := parseTestCase{
		raw:  "host:tcp4:localhost:5050",
		kind: Kind_Forwarding,
		expected: &URL{
			Kind:     Kind_Forwarding,
			Protocol: Protocol_SSH,
			User:     "",
			Host:     "host",
			Port:     0,
			Path:     "tcp4:localhost:5050",
		},
	}
	test.run(t)
}

func TestParseForwardingSCPSSHHostnameUnixDomainSocketEndpoint(t *testing.T) {
	test := parseTestCase{
		raw:  "host:unix:~/socket.sock",
		kind: Kind_Forwarding,
		expected: &URL{
			Kind:     Kind_Forwarding,
			Protocol: Protocol_SSH,
			User:     "",
			Host:     "host",
			Port:     0,
			Path:     "unix:~/socket.sock",
		},
	}
	test.run(t)
}

func TestParseForwardingSCPSSHHostnameNamedPipeEndpoint(t *testing.T) {
	test := parseTestCase{
		raw:  `host:npipe:\\.\pipe\pipename`,
		kind: Kind_Forwarding,
		expected: &URL{
			Kind:     Kind_Forwarding,
			Protocol: Protocol_SSH,
			User:     "",
			Host:     "host",
			Port:     0,
			Path:     `npipe:\\.\pipe\pipename`,
		},
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnamePath(t *testing.T) {
	test := parseTestCase{
		raw: "user@host:path",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "host",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnamePathWithColonInMiddle(t *testing.T) {
	test := parseTestCase{
		raw: "user@host:pa:th",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "host",
			Port:     0,
			Path:     "pa:th",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnamePathWithColonAtEnd(t *testing.T) {
	test := parseTestCase{
		raw: "user@host:path:",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "host",
			Port:     0,
			Path:     "path:",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnameWithAtPath(t *testing.T) {
	test := parseTestCase{
		raw: "user@ho@st:path",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "ho@st",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnamePathWithAt(t *testing.T) {
	test := parseTestCase{
		raw: "user@host:pa@th",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "host",
			Port:     0,
			Path:     "pa@th",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnamePortPath(t *testing.T) {
	test := parseTestCase{
		raw: "user@host:65535:path",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "host",
			Port:     65535,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnameZeroPortPath(t *testing.T) {
	test := parseTestCase{
		raw: "user@host:0:path",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "host",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnameDoubleZeroPortPath(t *testing.T) {
	test := parseTestCase{
		raw: "user@host:00:path",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "host",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnameOutOfBoundsPortInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:65536:path",
		fail: true,
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnameHexNumericPath(t *testing.T) {
	test := parseTestCase{
		raw: "user@host:aaa:path",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "host",
			Port:     0,
			Path:     "aaa:path",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUnicodeUsernameHostnamePath(t *testing.T) {
	test := parseTestCase{
		raw: "üsér@høst:пат",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "üsér",
			Host:     "høst",
			Port:     0,
			Path:     "пат",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUnicodeUsernameHostnamePortPath(t *testing.T) {
	test := parseTestCase{
		raw: "üsér@høst:23:пат",
		expected: &URL{
			Protocol: Protocol_SSH,
			User:     "üsér",
			Host:     "høst",
			Port:     23,
			Path:     "пат",
		},
	}
	test.run(t)
}

func TestParseForwardingDockerWithSourceSpecificVariables(t *testing.T) {
	test := parseTestCase{
		raw:   "docker://cøntainer:unix:/some/socket.sock",
		kind:  Kind_Forwarding,
		first: true,
		expected: &URL{
			Kind:     Kind_Forwarding,
			Protocol: Protocol_Docker,
			Host:     "cøntainer",
			Path:     "unix:/some/socket.sock",
			Environment: map[string]string{
				"DOCKER_HOST":       defaultDockerHost,
				"DOCKER_TLS_VERIFY": defaultDockerTLSVerify,
				"DOCKER_CONTEXT":    sourceSpecificDockerContext,
			},
		},
	}
	test.run(t)
}

func TestParseForwardingDockerWithDestinationSpecificVariables(t *testing.T) {
	test := parseTestCase{
		raw:  "docker://cøntainer:tcp6:[::1]:5543",
		kind: Kind_Forwarding,
		expected: &URL{
			Kind:     Kind_Forwarding,
			Protocol: Protocol_Docker,
			Host:     "cøntainer",
			Path:     "tcp6:[::1]:5543",
			Environment: map[string]string{
				"DOCKER_HOST":       defaultDockerHost,
				"DOCKER_TLS_VERIFY": destinationSpecificDockerTLSVerify,
			},
		},
	}
	test.run(t)
}

func TestParseDockerWithBetaSpecificVariables(t *testing.T) {
	test := parseTestCase{
		raw:  "docker://cøntainer/пат/to/the file",
		fail: false,
		expected: &URL{
			Protocol: Protocol_Docker,
			Host:     "cøntainer",
			Path:     "/пат/to/the file",
			Environment: map[string]string{
				"DOCKER_HOST":       defaultDockerHost,
				"DOCKER_TLS_VERIFY": betaSpecificDockerTLSVerify,
			},
		},
	}
	test.run(t)
}

func TestParseDockerWithWindowsPathAndAlphaSpecificVariables(t *testing.T) {
	test := parseTestCase{
		raw:   `docker://cøntainer/C:\пат/to\the file`,
		first: true,
		fail:  false,
		expected: &URL{
			Protocol: Protocol_Docker,
			Host:     "cøntainer",
			Path:     `C:\пат/to\the file`,
			Environment: map[string]string{
				"DOCKER_HOST":       alphaSpecificDockerHost,
				"DOCKER_TLS_VERIFY": defaultDockerTLSVerify,
			},
		},
	}
	test.run(t)
}

func TestParseDockerWithUsernameHomeRelativePathAndAlphaSpecificVariables(t *testing.T) {
	test := parseTestCase{
		raw:   "docker://üsér@cøntainer/~/пат/to/the file",
		first: true,
		fail:  false,
		expected: &URL{
			Protocol: Protocol_Docker,
			User:     "üsér",
			Host:     "cøntainer",
			Path:     "~/пат/to/the file",
			Environment: map[string]string{
				"DOCKER_HOST":       alphaSpecificDockerHost,
				"DOCKER_TLS_VERIFY": defaultDockerTLSVerify,
			},
		},
	}
	test.run(t)
}

func TestParseDockerWithUsernameUserRelativePathAndAlphaSpecificVariables(t *testing.T) {
	test := parseTestCase{
		raw:   "docker://üsér@cøntainer/~otheruser/пат/to/the file",
		first: true,
		fail:  false,
		expected: &URL{
			Protocol: Protocol_Docker,
			User:     "üsér",
			Host:     "cøntainer",
			Path:     "~otheruser/пат/to/the file",
			Environment: map[string]string{
				"DOCKER_HOST":       alphaSpecificDockerHost,
				"DOCKER_TLS_VERIFY": defaultDockerTLSVerify,
			},
		},
	}
	test.run(t)
}
