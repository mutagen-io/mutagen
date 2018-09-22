package url

import (
	"runtime"
	"testing"
)

type parseTestCase struct {
	raw      string
	alpha    bool
	fail     bool
	expected *URL
}

func (c *parseTestCase) run(t *testing.T) {
	// Attempt to parse.
	url, err := Parse(c.raw, c.alpha)
	if err != nil {
		if !c.fail {
			t.Fatal("parsing failed when it should have succeeded:", err)
		}
		return
	} else if c.fail {
		t.Fatal("parsing should have failed but did not")
	}

	// Verify protocol.
	if url.Protocol != c.expected.Protocol {
		t.Error("protocol mismatch:", url.Protocol, "!=", c.expected.Protocol)
	}

	// Verify username.
	if url.Username != c.expected.Username {
		t.Error("username mismatch:", url.Username, "!=", c.expected.Username)
	}

	// Verify hostname.
	if url.Hostname != c.expected.Hostname {
		t.Error("hostname mismatch:", url.Hostname, "!=", c.expected.Hostname)
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

func TestParseLocalPath(t *testing.T) {
	test := parseTestCase{
		raw: "/this/is/a:path",
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     "/this/is/a:path",
		},
	}
	test.run(t)
}

func TestParseLocalPathWithAtSymbol(t *testing.T) {
	test := parseTestCase{
		raw: "some@path",
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     "some@path",
		},
	}
	test.run(t)
}

func TestParsePOSIXSCPSSHWindowsLocal(t *testing.T) {
	expected := &URL{
		Protocol: Protocol_SSH,
		Hostname: "C",
		Path: "/local/path",
	}
	if runtime.GOOS == "windows" {
		expected = &URL{
			Path: "C:/local/path",
		}
	}
	test := &parseTestCase{
		raw: "C:/local/path",
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
			Username: "",
			Hostname: "host",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseSCPSSHUsernameHostnamePath(t *testing.T) {
	test := parseTestCase{
		raw: "user@host:path",
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
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
			Username: "user",
			Hostname: "host",
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
			Username: "user",
			Hostname: "host",
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
			Username: "user",
			Hostname: "ho@st",
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
			Username: "user",
			Hostname: "host",
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
			Username: "user",
			Hostname: "host",
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
			Username: "user",
			Hostname: "host",
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
			Username: "user",
			Hostname: "host",
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
			Username: "user",
			Hostname: "host",
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
			Username: "üsér",
			Hostname: "høst",
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
			Username: "üsér",
			Hostname: "høst",
			Port:     23,
			Path:     "пат",
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
			Hostname: "cøntainer",
			Path:     "/пат/to/the file",
			Environment: map[string]string{
				DockerHostEnvironmentVariable:      defaultDockerHost,
				DockerTLSVerifyEnvironmentVariable: betaSpecificDockerTLSVerify,
				DockerCertPathEnvironmentVariable:  "",
			},
		},
	}
	test.run(t)
}

func TestParseDockerWithWindowsPathAndAlphaSpecificVariables(t *testing.T) {
	test := parseTestCase{
		raw:   `docker://cøntainer/C:\пат/to\the file`,
		alpha: true,
		fail:  false,
		expected: &URL{
			Protocol: Protocol_Docker,
			Hostname: "cøntainer",
			Path:     `C:\пат/to\the file`,
			Environment: map[string]string{
				DockerHostEnvironmentVariable:      alphaSpecificDockerHost,
				DockerTLSVerifyEnvironmentVariable: defaultDockerTLSVerify,
				DockerCertPathEnvironmentVariable:  "",
			},
		},
	}
	test.run(t)
}

func TestParseDockerWithUsernameHomeRelativePathAndAlphaSpecificVariables(t *testing.T) {
	test := parseTestCase{
		raw:   "docker://üsér@cøntainer/~/пат/to/the file",
		alpha: true,
		fail:  false,
		expected: &URL{
			Protocol: Protocol_Docker,
			Username: "üsér",
			Hostname: "cøntainer",
			Path:     "~/пат/to/the file",
			Environment: map[string]string{
				DockerHostEnvironmentVariable:      alphaSpecificDockerHost,
				DockerTLSVerifyEnvironmentVariable: defaultDockerTLSVerify,
				DockerCertPathEnvironmentVariable:  "",
			},
		},
	}
	test.run(t)
}

func TestParseDockerWithUsernameUserRelativePathAndAlphaSpecificVariables(t *testing.T) {
	test := parseTestCase{
		raw:   "docker://üsér@cøntainer/~otheruser/пат/to/the file",
		alpha: true,
		fail:  false,
		expected: &URL{
			Protocol: Protocol_Docker,
			Username: "üsér",
			Hostname: "cøntainer",
			Path:     "~otheruser/пат/to/the file",
			Environment: map[string]string{
				DockerHostEnvironmentVariable:      alphaSpecificDockerHost,
				DockerTLSVerifyEnvironmentVariable: defaultDockerTLSVerify,
				DockerCertPathEnvironmentVariable:  "",
			},
		},
	}
	test.run(t)
}
