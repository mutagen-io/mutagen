package sync

import (
	"testing"
)

// TestSynchronizationModeUnmarshal tests that unmarshaling from a string
// specification succeeeds for SynchronizationMode.
func TestSynchronizationModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		text          string
		expectedMode  SynchronizationMode
		expectFailure bool
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
		if err := mode.UnmarshalText([]byte(testCase.text)); err != nil {
			if !testCase.expectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", testCase.text, err)
			}
		} else if testCase.expectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", testCase.text)
		} else if mode != testCase.expectedMode {
			t.Errorf(
				"unmarshaled mode (%s) does not match expected (%s)",
				mode,
				testCase.expectedMode,
			)
		}
	}
}

// TestSynchronizationModeSupported tests that SynchronizationMode support
// detection works as expected.
func TestSynchronizationModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            SynchronizationMode
		expectSupported bool
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
		if supported := testCase.mode.Supported(); supported != testCase.expectSupported {
			t.Errorf(
				"mode support status (%t) does not match expected (%t)",
				supported,
				testCase.expectSupported,
			)
		}
	}
}

// TestSynchronizationModeDescription tests that SynchronizationMode description
// generation works as expected.
func TestSynchronizationModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                SynchronizationMode
		expectedDescription string
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
		if description := testCase.mode.Description(); description != testCase.expectedDescription {
			t.Errorf(
				"mode description (%s) does not match expected (%s)",
				description,
				testCase.expectedDescription,
			)
		}
	}
}
