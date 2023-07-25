package ignore

import (
	"testing"
)

// TestIgnoreVCSModeIsDefault tests IgnoreVCSMode.IsDefault.
func TestIgnoreVCSModeIsDefault(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value    IgnoreVCSMode
		expected bool
	}{
		{IgnoreVCSMode_IgnoreVCSModeDefault - 1, false},
		{IgnoreVCSMode_IgnoreVCSModeDefault, true},
		{IgnoreVCSMode_IgnoreVCSModeIgnore, false},
		{IgnoreVCSMode_IgnoreVCSModePropagate, false},
		{IgnoreVCSMode_IgnoreVCSModePropagate + 1, false},
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

// TestIgnoreVCSModeUnmarshalText tests IgnoreVCSMode.UnmarshalText.
func TestIgnoreVCSModeUnmarshalText(t *testing.T) {
	// Define test cases.
	tests := []struct {
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
	for _, test := range tests {
		var mode IgnoreVCSMode
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
