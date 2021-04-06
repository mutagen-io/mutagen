package core

import (
	"testing"
)

// TestSymbolicLinkModeIsDefault tests SymbolicLinkMode.IsDefault.
func TestSymbolicLinkModeIsDefault(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value    SymbolicLinkMode
		expected bool
	}{
		{SymbolicLinkMode_SymbolicLinkModeDefault - 1, false},
		{SymbolicLinkMode_SymbolicLinkModeDefault, true},
		{SymbolicLinkMode_SymbolicLinkModeIgnore, false},
		{SymbolicLinkMode_SymbolicLinkModePortable, false},
		{SymbolicLinkMode_SymbolicLinkModePOSIXRaw, false},
		{SymbolicLinkMode_SymbolicLinkModePOSIXRaw + 1, false},
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

// TestSymbolicLinkModeUnmarshalText tests SymbolicLinkMode.UnmarshalText.
func TestSymbolicLinkModeUnmarshalText(t *testing.T) {
	// Define test cases.
	tests := []struct {
		text          string
		expectedMode  SymbolicLinkMode
		expectFailure bool
	}{
		{"", SymbolicLinkMode_SymbolicLinkModeDefault, true},
		{"asdf", SymbolicLinkMode_SymbolicLinkModeDefault, true},
		{"ignore", SymbolicLinkMode_SymbolicLinkModeIgnore, false},
		{"portable", SymbolicLinkMode_SymbolicLinkModePortable, false},
		{"posix-raw", SymbolicLinkMode_SymbolicLinkModePOSIXRaw, false},
	}

	// Process test cases.
	for _, test := range tests {
		var mode SymbolicLinkMode
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

// TestSymbolicLinkModeSupported tests SymbolicLinkMode.Supported.
func TestSymbolicLinkModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            SymbolicLinkMode
		expectSupported bool
	}{
		{SymbolicLinkMode_SymbolicLinkModeDefault, false},
		{SymbolicLinkMode_SymbolicLinkModeIgnore, true},
		{SymbolicLinkMode_SymbolicLinkModePortable, true},
		{SymbolicLinkMode_SymbolicLinkModePOSIXRaw, true},
		{(SymbolicLinkMode_SymbolicLinkModePOSIXRaw + 1), false},
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

// TestSymbolicLinkModeDescription tests SymbolicLinkMode.Description.
func TestSymbolicLinkModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                SymbolicLinkMode
		expectedDescription string
	}{
		{SymbolicLinkMode_SymbolicLinkModeDefault, "Default"},
		{SymbolicLinkMode_SymbolicLinkModeIgnore, "Ignore"},
		{SymbolicLinkMode_SymbolicLinkModePortable, "Portable"},
		{SymbolicLinkMode_SymbolicLinkModePOSIXRaw, "POSIX Raw"},
		{(SymbolicLinkMode_SymbolicLinkModePOSIXRaw + 1), "Unknown"},
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
