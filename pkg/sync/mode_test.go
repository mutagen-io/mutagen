package sync

import (
	"testing"
)

// TestSynchronizationModeUnmarshal tests that unmarshaling from a string
// specification succeeeds for SynchronizationMode.
func TestSynchronizationModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  SynchronizationMode
		ExpectFailure bool
	}{
		{"", SynchronizationMode_SynchronizationModeDefault, true},
		{"asdf", SynchronizationMode_SynchronizationModeDefault, true},
		{"two-way-safe", SynchronizationMode_SynchronizationModeTwoWaySafe, false},
		{"two-way-resolved", SynchronizationMode_SynchronizationModeTwoWayResolved, false},
		{"one-way-safe", SynchronizationMode_SynchronizationModeOneWaySafe, false},
		{"one-way-replica", SynchronizationMode_SynchronizationModeOneWayReplica, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode SynchronizationMode
		if err := mode.UnmarshalText([]byte(testCase.Text)); err != nil {
			if !testCase.ExpectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", testCase.Text, err)
			}
		} else if testCase.ExpectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", testCase.Text)
		} else if mode != testCase.ExpectedMode {
			t.Errorf(
				"unmarshaled mode (%s) does not match expected (%s)",
				mode,
				testCase.ExpectedMode,
			)
		}
	}
}

// TestSynchronizationModeSupported tests that SynchronizationMode support
// detection works as expected.
func TestSynchronizationModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            SynchronizationMode
		ExpectSupported bool
	}{
		{SynchronizationMode_SynchronizationModeDefault, false},
		{SynchronizationMode_SynchronizationModeTwoWaySafe, true},
		{SynchronizationMode_SynchronizationModeTwoWayResolved, true},
		{SynchronizationMode_SynchronizationModeOneWaySafe, true},
		{SynchronizationMode_SynchronizationModeOneWayReplica, true},
		{(SynchronizationMode_SynchronizationModeOneWayReplica + 1), false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if supported := testCase.Mode.Supported(); supported != testCase.ExpectSupported {
			t.Errorf(
				"mode support status (%t) does not match expected (%t)",
				supported,
				testCase.ExpectSupported,
			)
		}
	}
}

// TestSynchronizationModeDescription tests that SynchronizationMode description
// generation works as expected.
func TestSynchronizationModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                SynchronizationMode
		ExpectedDescription string
	}{
		{SynchronizationMode_SynchronizationModeDefault, "Default"},
		{SynchronizationMode_SynchronizationModeTwoWaySafe, "Two Way Safe"},
		{SynchronizationMode_SynchronizationModeTwoWayResolved, "Two Way Resolved"},
		{SynchronizationMode_SynchronizationModeOneWaySafe, "One Way Safe"},
		{SynchronizationMode_SynchronizationModeOneWayReplica, "One Way Replica"},
		{(SynchronizationMode_SynchronizationModeOneWayReplica + 1), "Unknown"},
	}

	// Process test cases.
	for _, testCase := range testCases {
		if description := testCase.Mode.Description(); description != testCase.ExpectedDescription {
			t.Errorf(
				"mode description (%s) does not match expected (%s)",
				description,
				testCase.ExpectedDescription,
			)
		}
	}
}
