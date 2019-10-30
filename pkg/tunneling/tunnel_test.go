package tunneling

import (
	"testing"
)

// TestSupportedRoles verifies that tunnel role support is as expected.
func TestSupportedRoles(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		role     Role
		expected bool
	}{
		{Role_Host, true},
		{Role_Client, true},
		{Role_Client + 1, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if supported := testCase.role.Supported(); supported != testCase.expected {
			t.Errorf(
				"tunnel role (%s) support does not match expected: %t != %t",
				testCase.role,
				supported,
				testCase.expected,
			)
		}
	}
}

// TODO: Implement tests.
