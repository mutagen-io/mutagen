package forwarding

import (
	"testing"
)

// TestSupportedVersions verifies that session version support is as expected.
func TestSupportedVersions(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		version  Version
		expected bool
	}{
		{Version_Invalid, false},
		{Version_Version1, true},
		{Version_Version1 + 1, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if supported := testCase.version.Supported(); supported != testCase.expected {
			t.Errorf(
				"session version (%s) support does not match expected: %t != %t",
				testCase.version,
				supported,
				testCase.expected,
			)
		}
	}
}
