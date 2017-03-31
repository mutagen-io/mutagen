package url

import (
	"testing"
)

func TestParseWindowsPath(t *testing.T) {
	test := testCase{
		raw:  `C:\something`,
		fail: false,
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     `C:\something`,
		},
	}
	test.run(t)
}

func TestParseWindowsPathForward(t *testing.T) {
	test := testCase{
		raw:  "C:/something",
		fail: false,
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     "C:/something",
		},
	}
	test.run(t)
}

func TestParseWindowsPathSmall(t *testing.T) {
	test := testCase{
		raw:  `c:\something`,
		fail: false,
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     `c:\something`,
		},
	}
	test.run(t)
}
