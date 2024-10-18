package forwarding

import (
	"testing"
)

// TestStatusUnmarshal tests that unmarshaling from a string specification
// succeeeds for Status.
func TestStatusUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		text          string
		expected      Status
		expectFailure bool
	}{
		{"", Status_Disconnected, true},
		{"asdf", Status_Disconnected, true},
		{"disconnected", Status_Disconnected, false},
		{"connecting-source", Status_ConnectingSource, false},
		{"connecting-destination", Status_ConnectingDestination, false},
		{"forwarding", Status_ForwardingConnections, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var status Status
		if err := status.UnmarshalText([]byte(testCase.text)); err != nil {
			if !testCase.expectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", testCase.text, err)
			}
		} else if testCase.expectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", testCase.text)
		} else if status != testCase.expected {
			t.Errorf(
				"unmarshaled status (%s) does not match expected (%s)",
				status,
				testCase.expected,
			)
		}
	}
}
