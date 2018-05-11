package url

import (
	"testing"
)

type formatTestCase struct {
	url      *URL
	expected string
}

func (c *formatTestCase) run(t *testing.T) {
	formatted := c.url.Format()
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
	url.Format()
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
