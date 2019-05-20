package sync

import (
	"testing"
)

// TestIgnoreVCSModeUnmarshal tests that unmarshaling from a string
// specification succeeeds for IgnoreVCSMode.
func TestIgnoreVCSModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		text          string
		expectedMode  IgnoreVCSMode
		expectFailure bool
	}{
		{"", IgnoreVCSMode_IgnoreVCSModeDefault, true},
		{"asdf", IgnoreVCSMode_IgnoreVCSModeDefault, true},
		{"true", IgnoreVCSMode_IgnoreVCSModeIgnore, false},
		{"false", IgnoreVCSMode_IgnoreVCSModePropagate, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode IgnoreVCSMode
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

// TestIgnoreVCSModeSupported tests that IgnoreVCSMode support detection works
// as expected.
func TestIgnoreVCSModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            IgnoreVCSMode
		expectSupported bool
	}{
		{IgnoreVCSMode_IgnoreVCSModeDefault, false},
		{IgnoreVCSMode_IgnoreVCSModeIgnore, true},
		{IgnoreVCSMode_IgnoreVCSModePropagate, true},
		{(IgnoreVCSMode_IgnoreVCSModePropagate + 1), false},
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

// TestIgnoreVCSModeDescription tests that IgnoreVCSMode description generation
// works as expected.
func TestIgnoreVCSModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                IgnoreVCSMode
		expectedDescription string
	}{
		{IgnoreVCSMode_IgnoreVCSModeDefault, "Default"},
		{IgnoreVCSMode_IgnoreVCSModeIgnore, "Ignore"},
		{IgnoreVCSMode_IgnoreVCSModePropagate, "Propagate"},
		{(IgnoreVCSMode_IgnoreVCSModePropagate + 1), "Unknown"},
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
