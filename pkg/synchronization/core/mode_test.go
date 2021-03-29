package core

import (
	"testing"
)

// TestSynchronizationModeIsDefault tests SynchronizationMode.IsDefault.
func TestSynchronizationModeIsDefault(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value    SynchronizationMode
		expected bool
	}{
		{SynchronizationMode_SynchronizationModeDefault - 1, false},
		{SynchronizationMode_SynchronizationModeDefault, true},
		{SynchronizationMode_SynchronizationModeTwoWaySafe, false},
		{SynchronizationMode_SynchronizationModeTwoWayResolved, false},
		{SynchronizationMode_SynchronizationModeOneWaySafe, false},
		{SynchronizationMode_SynchronizationModeOneWayReplica, false},
		{SynchronizationMode_SynchronizationModeOneWayReplica + 1, false},
	}

	// Process test cases.
	for i, test := range tests {
		if result := test.value.IsDefault(); result && !test.expected {
			t.Errorf("test index %d: value was unexpectedly classified as default", i)
		} else if !result && test.expected {
			t.Errorf("test index %d: value was unexpectedly classified as non-default", i)
		}
	}
}

// TestSynchronizationModeUnmarshalText tests SynchronizationMode.UnmarshalText.
func TestSynchronizationModeUnmarshalText(t *testing.T) {
	// Define test cases.
	tests := []struct {
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
	for _, test := range tests {
		var mode SynchronizationMode
		if err := mode.UnmarshalText([]byte(test.text)); err != nil {
			if !test.expectFailure {
				t.Errorf("unable to unmarshal text (%s): %s", test.text, err)
			}
		} else if test.expectFailure {
			t.Error("unmarshaling succeeded unexpectedly for text:", test.text)
		} else if mode != test.expectedMode {
			t.Errorf(
				"unmarshaled mode (%s) does not match expected (%s)",
				mode,
				test.expectedMode,
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
