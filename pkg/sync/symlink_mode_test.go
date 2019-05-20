package sync

import (
	"testing"
)

// TestSymlinkModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for SymlinkMode.
func TestSymlinkModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
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
	for _, testCase := range testCases {
		var mode SymlinkMode
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
