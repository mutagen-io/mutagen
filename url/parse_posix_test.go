// +build !windows

package url

import (
	"testing"
)

func TestParseWindowsPath(t *testing.T) {
	test := testCase{
		raw:  `C:\something`,
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "",
			Hostname: "C",
			Port:     0,
			Path:     `\something`,
		},
	}
	test.run(t)
}

func TestParseWindowsPathForward(t *testing.T) {
	test := testCase{
		raw:  "C:/something",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "",
			Hostname: "C",
			Port:     0,
			Path:     "/something",
		},
	}
	test.run(t)
}

func TestParseWindowsPathSmall(t *testing.T) {
	test := testCase{
		raw:  `c:\something`,
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "",
			Hostname: "c",
			Port:     0,
			Path:     `\something`,
		},
	}
	test.run(t)
}
