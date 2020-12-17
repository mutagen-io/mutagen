package url

import (
	"testing"
)

type formatTestCase struct {
	url               *URL
	environmentPrefix string
	expected          string
}

func (c *formatTestCase) run(t *testing.T) {
	formatted := c.url.Format(c.environmentPrefix)
	if formatted != c.expected {
		t.Fatal("formatting mismatch:", formatted, "!=", c.expected)
	}
}

func TestFormatInvalidProtocol(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("formatting invalid protocol did not panic")
		}
	}()
	url := &URL{Protocol: Protocol(-1)}
	url.Format("")
}

func TestFormatLocal(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_Local,
			Path:     "/test/path",
		},
		expected: "/test/path",
	}
	test.run(t)
}

func TestFormatForwardingLocal(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Kind:     Kind_Forwarding,
			Protocol: Protocol_Local,
			Path:     "/test/path",
		},
		expected: "/test/path",
	}
	test.run(t)
}

func TestFormatSSHHostnamePath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_SSH,
			User:     "",
			Host:     "host",
			Port:     0,
			Path:     "/test/path",
		},
		expected: "host:/test/path",
	}
	test.run(t)
}

func TestFormatForwardingSSHHostnamePath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_SSH,
			User:     "",
			Host:     "host",
			Port:     0,
			Path:     "tcp:localhost:6060",
		},
		expected: "host:tcp:localhost:6060",
	}
	test.run(t)
}

func TestFormatSSHUsernameHostnamePath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "host",
			Port:     0,
			Path:     "/test/path",
		},
		expected: "user@host:/test/path",
	}
	test.run(t)
}

func TestFormatSSHHostnamePortPath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_SSH,
			User:     "",
			Host:     "host",
			Port:     23,
			Path:     "/test/path",
		},
		expected: "host:23:/test/path",
	}
	test.run(t)
}

func TestFormatSSHUsernameHostnamePortPath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_SSH,
			User:     "user",
			Host:     "host",
			Port:     23,
			Path:     "/test/path",
		},
		expected: "user@host:23:/test/path",
	}
	test.run(t)
}

func TestFormatDockerInvalidEmptyPath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_Docker,
			Host:     "container",
			Path:     "",
		},
		expected: invalidDockerURLFormat,
	}
	test.run(t)
}

func TestFormatDockerInvalidBadFirstPathCharacter(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_Docker,
			Host:     "container",
			Path:     "$5",
		},
		expected: invalidDockerURLFormat,
	}
	test.run(t)
}

func TestFormatDocker(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_Docker,
			Host:     "container",
			Path:     "/test/path/to/the file",
			Environment: map[string]string{
				"DOCKER_HOST": "unix:///path/to/docker.sock",
			},
		},
		environmentPrefix: "|",
		expected:          "docker://container/test/path/to/the file|DOCKER_HOST=unix:///path/to/docker.sock",
	}
	test.run(t)
}

func TestFormatForwardingDocker(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Kind:     Kind_Forwarding,
			Protocol: Protocol_Docker,
			Host:     "container",
			Path:     "tcp4:localhost:8080",
			Environment: map[string]string{
				"DOCKER_HOST": "unix:///path/to/docker.sock",
			},
		},
		environmentPrefix: "|",
		expected:          "docker://container:tcp4:localhost:8080|DOCKER_HOST=unix:///path/to/docker.sock",
	}
	test.run(t)
}

func TestFormatDockerWithUsernameAndHomeRelativePath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_Docker,
			User:     "user",
			Host:     "container",
			Path:     "~/test/path/to/the file",
			Environment: map[string]string{
				"DOCKER_HOST":       "unix:///path/to/docker.sock",
				"DOCKER_TLS_VERIFY": "true",
			},
		},
		environmentPrefix: "|",
		expected:          "docker://user@container/~/test/path/to/the file|DOCKER_HOST=unix:///path/to/docker.sock|DOCKER_TLS_VERIFY=true",
	}
	test.run(t)
}

func TestFormatDockerWithUsernameAndUserRelativePath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_Docker,
			User:     "user",
			Host:     "container",
			Path:     "~otheruser/test/path/to/the file",
			Environment: map[string]string{
				"DOCKER_HOST":       "unix:///path/to/docker.sock",
				"DOCKER_TLS_VERIFY": "true",
			},
		},
		environmentPrefix: "|",
		expected:          "docker://user@container/~otheruser/test/path/to/the file|DOCKER_HOST=unix:///path/to/docker.sock|DOCKER_TLS_VERIFY=true",
	}
	test.run(t)
}

func TestFormatDockerWithWindowsPathPath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_Docker,
			Host:     "container",
			Path:     `C:\A\Windows\File Path `,
			Environment: map[string]string{
				"DOCKER_HOST":       "unix:///path/to/docker.sock",
				"DOCKER_TLS_VERIFY": "true",
			},
		},
		environmentPrefix: "|",
		expected:          `docker://container/C:\A\Windows\File Path |DOCKER_HOST=unix:///path/to/docker.sock|DOCKER_TLS_VERIFY=true`,
	}
	test.run(t)
}
