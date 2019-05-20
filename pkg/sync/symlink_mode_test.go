package sync

import (
	"testing"
)

// TestSymlinkModeUnmarshal tests that unmarshaling from a string specification
// succeeeds for SymlinkMode.
func TestSymlinkModeUnmarshal(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Text          string
		ExpectedMode  SymlinkMode
		ExpectFailure bool
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

// TestSymlinkModeSupported tests that SymlinkMode support detection works as
// expected.
func TestSymlinkModeSupported(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode            SymlinkMode
		ExpectSupported bool
	}{
		{SymlinkMode_SymlinkModeDefault, false},
		{SymlinkMode_SymlinkModeIgnore, true},
		{SymlinkMode_SymlinkModePortable, true},
		{SymlinkMode_SymlinkModePOSIXRaw, true},
		{(SymlinkMode_SymlinkModePOSIXRaw + 1), false},
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

// TestSymlinkModeDescription tests that SymlinkMode description generation
// works as expected.
func TestSymlinkModeDescription(t *testing.T) {
	// Set up test cases.
	testCases := []struct {
		Mode                SymlinkMode
		ExpectedDescription string
	}{
		{SymlinkMode_SymlinkModeDefault, "Default"},
		{SymlinkMode_SymlinkModeIgnore, "Ignore"},
		{SymlinkMode_SymlinkModePortable, "Portable"},
		{SymlinkMode_SymlinkModePOSIXRaw, "POSIX Raw"},
		{(SymlinkMode_SymlinkModePOSIXRaw + 1), "Unknown"},
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
