package core

import (
	"testing"
)

// TestSymlinkModeIsDefault tests SymlinkMode.IsDefault.
func TestSymlinkModeIsDefault(t *testing.T) {
	// Define test cases.
	tests := []struct {
		value    SymlinkMode
		expected bool
	}{
		{SymlinkMode_SymlinkModeDefault - 1, false},
		{SymlinkMode_SymlinkModeDefault, true},
		{SymlinkMode_SymlinkModeIgnore, false},
		{SymlinkMode_SymlinkModePortable, false},
		{SymlinkMode_SymlinkModePOSIXRaw, false},
		{SymlinkMode_SymlinkModePOSIXRaw + 1, false},
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

// TestSymlinkModeUnmarshalText tests SymlinkMode.UnmarshalText.
func TestSymlinkModeUnmarshalText(t *testing.T) {
	// Define test cases.
	tests := []struct {
		text          string
		expectedMode  SymlinkMode
		expectFailure bool
	}{
		{"", SymlinkMode_SymlinkModeDefault, true},
		{"asdf", SymlinkMode_SymlinkModeDefault, true},
		{"ignore", SymlinkMode_SymlinkModeIgnore, false},
		{"portable", SymlinkMode_SymlinkModePortable, false},
		{"posix-raw", SymlinkMode_SymlinkModePOSIXRaw, false},
	}

	// Process test cases.
	for _, test := range tests {
		var mode SymlinkMode
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

// TestSymlinkModeSupported tests that SymlinkMode support detection works as
// expected.
func TestSymlinkModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode            SymlinkMode
		expectSupported bool
	}{
		{SymlinkMode_SymlinkModeDefault, false},
		{SymlinkMode_SymlinkModeIgnore, true},
		{SymlinkMode_SymlinkModePortable, true},
		{SymlinkMode_SymlinkModePOSIXRaw, true},
		{(SymlinkMode_SymlinkModePOSIXRaw + 1), false},
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

// TestSymlinkModeDescription tests that SymlinkMode description generation
// works as expected.
func TestSymlinkModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		mode                SymlinkMode
		expectedDescription string
	}{
		{SymlinkMode_SymlinkModeDefault, "Default"},
		{SymlinkMode_SymlinkModeIgnore, "Ignore"},
		{SymlinkMode_SymlinkModePortable, "Portable"},
		{SymlinkMode_SymlinkModePOSIXRaw, "POSIX Raw"},
		{(SymlinkMode_SymlinkModePOSIXRaw + 1), "Unknown"},
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
