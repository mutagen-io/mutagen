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

func TestFormatSSHHostnamePath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_SSH,
			Username: "",
			Hostname: "host",
			Port:     0,
			Path:     "/test/path",
		},
		expected: "host:/test/path",
	}
	test.run(t)
}

func TestFormatSSHUsernameHostnamePath(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
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
			Username: "",
			Hostname: "host",
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
			Username: "user",
			Hostname: "host",
			Port:     23,
			Path:     "/test/path",
		},
		expected: "user@host:23:/test/path",
	}
	test.run(t)
}

func TestFormatDocker(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_Docker,
			Hostname: "container",
			Path:     "/test/path/to/the file",
			Environment: map[string]string{
				DockerHostEnvironmentVariable: "unix:///path/to/docker.sock",
			},
		},
		environmentPrefix: "|",
		expected:          "docker:container:/test/path/to/the file|DOCKER_HOST=unix:///path/to/docker.sock|DOCKER_TLS_VERIFY=|DOCKER_CERT_PATH=",
	}
	test.run(t)
}

func TestFormatDockerWithUsername(t *testing.T) {
	test := &formatTestCase{
		url: &URL{
			Protocol: Protocol_Docker,
			Username: "user",
			Hostname: "container",
			Path:     "/test/path/to/the file",
			Environment: map[string]string{
				DockerHostEnvironmentVariable:      "unix:///path/to/docker.sock",
				DockerTLSVerifyEnvironmentVariable: "true",
			},
		},
		environmentPrefix: "|",
		expected:          "docker:user@container:/test/path/to/the file|DOCKER_HOST=unix:///path/to/docker.sock|DOCKER_TLS_VERIFY=|DOCKER_CERT_PATH=",
	}
	test.run(t)
}
