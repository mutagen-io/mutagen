package sync

import (
	"testing"
)

// TestIgnoreVCSModeUnmarshal tests that unmarshaling from a string
// specification succeeeds for IgnoreVCSMode.
func TestIgnoreVCSModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  IgnoreVCSMode
		ExpectFailure bool
	}{
		{"", IgnoreVCSMode_IgnoreVCSModeDefault, true},
		{"asdf", IgnoreVCSMode_IgnoreVCSModeDefault, true},
		{"true", IgnoreVCSMode_IgnoreVCSModeIgnore, false},
		{"false", IgnoreVCSMode_IgnoreVCSModePropagate, false},
	}

	// Process test cases.
	for _, testCase := range testCases {
		var mode IgnoreVCSMode
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

// TestIgnoreVCSModeSupported tests that IgnoreVCSMode support detection works
// as expected.
func TestIgnoreVCSModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            IgnoreVCSMode
		ExpectSupported bool
	}{
		{IgnoreVCSMode_IgnoreVCSModeDefault, false},
		{IgnoreVCSMode_IgnoreVCSModeIgnore, true},
		{IgnoreVCSMode_IgnoreVCSModePropagate, true},
		{(IgnoreVCSMode_IgnoreVCSModePropagate + 1), false},
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

// TestIgnoreVCSModeDescription tests that IgnoreVCSMode description generation
// works as expected.
func TestIgnoreVCSModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                IgnoreVCSMode
		ExpectedDescription string
	}{
		{IgnoreVCSMode_IgnoreVCSModeDefault, "Default"},
		{IgnoreVCSMode_IgnoreVCSModeIgnore, "Ignore"},
		{IgnoreVCSMode_IgnoreVCSModePropagate, "Propagate"},
		{(IgnoreVCSMode_IgnoreVCSModePropagate + 1), "Unknown"},
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
