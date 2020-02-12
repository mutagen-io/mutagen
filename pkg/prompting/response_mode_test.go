package prompting

import (
	"testing"
)

// TestDetermineResponseMode tests determineResponseMode.
func TestDetermineResponseMode(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		prompt   string
		expected ResponseMode
	}{
		{"Question? (yes/no)? ", ResponseModeEcho},
		{"Question? (yes/no): ", ResponseModeEcho},
		{"Question? (yes/no/[fingerprint])? ", ResponseModeEcho},
		{"Please type 'yes', 'no' or the fingerprint: ", ResponseModeEcho},
		{"Give me your password: ", ResponseModeSecret},
	}

	// Perform tests.
	for _, testCase := range testCases {
		if mode := determineResponseMode(testCase.prompt); mode != testCase.expected {
			t.Errorf("prompt ('%s') response mode does not match expected: %v != %v", testCase.prompt, mode, testCase.expected)
		}
	}
}
