package forwarding

import (
	"testing"
)

// TestParse tests that the Parse function behaves as expected for a variety of
// test cases.
func TestParse(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		url              string
		expectedProtocol string
		expectedAddress  string
		expectFailure    bool
	}{
		{"", "", "", true},
		{"a", "", "", true},
		{"a:b", "", "", true},
		{"invalid::3992", "", "", true},
		{"tcp::3992", "tcp", ":3992", false},
		{"tcp4:localhost:3992", "tcp4", "localhost:3992", false},
		{"tcp6:[::1]:3992", "tcp6", "[::1]:3992", false},
		{"unix:/some/socket.sock", "unix", "/some/socket.sock", false},
		{`npipe:\\.\pipe\pipe_name`, "npipe", `\\.\pipe\pipe_name`, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		// Perform parsing and ensure that failure behavior is as expected.
		protocol, address, err := Parse(testCase.url)
		if err != nil {
			if !testCase.expectFailure {
				t.Errorf("parse failed for URL (%s): %v", testCase.url, err)
			}
			continue
		} else if testCase.expectFailure {
			t.Error("parse succeeded unexpectedly for URL:", testCase.url)
			continue
		}

		// Check that the protocol is what's expected.
		if protocol != testCase.expectedProtocol {
			t.Error("protocol does not match expected:", protocol, "!=", testCase.expectedProtocol)
		}

		// Check that the address is what's expected.
		if address != testCase.expectedAddress {
			t.Error("address does not match expected:", address, "!=", testCase.expectedAddress)
		}
	}
}
